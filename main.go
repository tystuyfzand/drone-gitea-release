package main

import (
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/sdk/gitea"
	"fmt"
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
		Target:  os.Getenv("DRONE_COMMIT"),
		Title:   parseEnvOrFile("PLUGIN_TITLE"),
		Note:    parseEnvOrFile("PLUGIN_BODY"),
	}

	release, err := client.CreateRelease(namespace, repo, opt)

	if err != nil {
		log.Fatalln("Error creating release:", err)
	}

	fmt.Println("Release created:", release.ID)

	defer func() {
		if r := recover(); r != nil {
			// Cleanup release if we failed to upload files so we can try again later
			err := client.DeleteRelease(namespace, repo, release.ID)

			if err != nil {
				log.Fatalln("Unable to delete release:", err)
			}

			log.Fatalln("Fatal error creating release:", r)
		}
	}()

	files := parseFiles()

	fmt.Println("Uploading files " + strings.Join(files, ", "))

	for _, file := range files {
		f, err := os.Open(file)

		if err != nil {
			fmt.Println("Unable to open file " + file + ": " + err.Error())
			continue
		}

		fmt.Println("Attaching file " + file)

		attachment, err := client.CreateReleaseAttachment(namespace, repo, release.ID, f, path.Base(file))

		f.Close()

		if err != nil {
			fmt.Println("Unable to open file " + file + ": " + err.Error())
			continue
		}

		fmt.Println("Attached " + attachment.Name)
	}
}

func parseEnvOrFile(name string) string {
	fileEnv := os.Getenv(name + "_FILE")

	if fileEnv != "" {
		b, err := ioutil.ReadFile(fileEnv)

		if err == nil {
			return strings.TrimSpace(string(b))
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

	// Attempt to resolve the server from various env variables

	envs := []string{
		"DRONE_REPO_LINK",
		"DRONE_GIT_HTTP_URL",
	}

	for _, env := range envs {
		server = os.Getenv(env)

		if server != "" {
			break
		}
	}

	if server == "" {
		log.Fatalln("Unable to find server in env variables")
	}

	u, err := url.Parse(server)

	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Fatalln("Unable to parse env for gitea url")
	}

	u.Path = ""

	return u.String()
}
