package main

import (
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/sdk/gitea"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func main() {
	server := parseServerFromRepo()
	token := os.Getenv("PLUGIN_API_KEY")

	namespace := os.Getenv("DRONE_REPO_NAMESPACE")
	repo := os.Getenv("DRONE_REPO_NAME")

	client := gitea.NewClient(server, token)

	opt := structs.CreateReleaseOption{
		TagName: parseEnvOrFile("PLUGIN_TAG"),
		Target: os.Getenv("DRONE_COMMIT"),
		Title: parseEnvOrFile("PLUGIN_TITLE"),
		Note: parseEnvOrFile("PLUGIN_BODY"),
	}

	release, err := client.CreateRelease(namespace, repo, opt)

	if err != nil {
		log.Fatalln("Error creating release:", err)
	}

	log.Println("Release created:", release.URL)

	defer func() {
		if r := recover(); r != nil {
			// Cleanup release if we failed to upload files so we can try again later
			err := client.DeleteRelease(namespace, repo, release.ID)

			if err != nil {
				log.Fatalln("Unable to delete release:", err)
			}
		}
	}()

	files := parseFiles()

	for _, file := range files {
		f, err := os.Open(file)

		if err != nil {
			continue
		}

		attachment, err := client.CreateReleaseAttachment(namespace, repo, release.ID, f, path.Base(file))

		if err != nil {
			panic("Unable to attach file: " + err.Error())
		}

		log.Println("Attached " + attachment.Name)
	}
}

func parseEnvOrFile(name string) string {
	fileEnv := os.Getenv(name + "_FILE")

	if fileEnv != "" {
		b, err := ioutil.ReadFile(fileEnv)

		if err == nil {
			return string(b)
		}
	}

	return os.Getenv(name)
}

func parseFiles() []string {
	split := strings.Split(os.Getenv("PLUGIN_FILES"), ",")

	var files []string

	for _, p := range split {
		globed, err := filepath.Glob(p)

		if err != nil {
			panic("Unable to glob " + p + ": " + err.Error())
		}

		if globed != nil {
			files = append(files, globed...)
		}
	}

	return files
}

func parseServerFromRepo() string {
	server := os.Getenv("PLUGIN_GITEA_SERVER")

	if server != "" {
		return server
	}

	u, err := url.Parse(os.Getenv("DRONE_REPO_LINK"))

	if err != nil {
		log.Fatalln("Unable to parse DRONE_REPO_LINK for gitea url")
	}

	u.Path = ""

	return u.String()
}