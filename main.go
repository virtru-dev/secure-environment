package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

	"regexp"

	"gopkg.in/urfave/cli.v1"
)

var envFileLineRegex = regexp.MustCompile("^([A-Za-z][0-9A-Za-z_])*=(.*)")

func loadEnvironment(data []string, getKeyVal func(item string) (key, val string)) map[string]string {
	items := make(map[string]string)
	for _, item := range data {
		key, val := getKeyVal(item)
		items[key] = val
	}
	return items
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	log.SetOutput(os.Stderr)
	log.SetLevel(log.WarnLevel)

	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Name = "secure-environment"
	app.Before = func(c *cli.Context) error {
		debugOn := c.Bool("debug")

		if debugOn {
			log.SetLevel(log.DebugLevel)
			log.Debug("secure-environment debug logging is on.")
		}

		return nil
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "Set debug logging on",
			EnvVar: "SECURE_ENVIRONMENT_DEBUG",
		},
	}

	flags := []cli.Flag{
		cli.StringFlag{
			Name:   "key",
			Usage:  "Sets the key arn",
			EnvVar: "SECURE_ENVIRONMENT_KEY",
		},
		cli.StringFlag{
			Name:   "url",
			Usage:  "url to the environment file",
			EnvVar: "SECURE_ENVIRONMENT_URL",
		},
		cli.StringFlag{
			Name:   "env-type",
			Value:  "envfile",
			Usage:  "content type of the environment file",
			EnvVar: "SECURE_ENVIRONMENT_TYPE",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "export",
			Usage:  "Create a bash compatible export output via stdout",
			Action: exportEnv,
			Flags:  flags,
		},
		{
			Name:   "import",
			Usage:  "Transforms an env file into encrypted env",
			Action: importEnv,
			Flags:  flags,
		},
	}

	app.Run(os.Args)
}

func importEnv(c *cli.Context) error {
	secureEnvironmentURL := c.String("url")
	secureEnvironmentKey := c.String("key")
	secureEnvironmentType := c.String("env-type")

	if secureEnvironmentURL == "" || secureEnvironmentKey == "" || secureEnvironmentType == "" {
		log.Debug("Missing required environment")
		return fmt.Errorf("Missing required environment variables")
	}

	outputFile, err := os.Create(c.Args().Get(1))

	defer outputFile.Close()

	if err != nil {
		return err
	}

	if file, err := os.Open(c.Args().Get(0)); err == nil {
		defer file.Close()

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		cipher, err := NewCipher()
		if err != nil {
			return nil
		}

		encryptedEnvelope, err := cipher.Encrypt(secureEnvironmentKey, fileBytes)
		if err != nil {
			return err
		}

		return s3PutObject(secureEnvironmentURL, encryptedEnvelope)
	}
	return nil
}

func exportEnv(c *cli.Context) error {
	secureEnvironmentURL := c.String("url")
	secureEnvironmentKey := c.String("key")
	secureEnvironmentType := c.String("env-type")

	if secureEnvironmentURL == "" || secureEnvironmentKey == "" || secureEnvironmentType == "" {
		log.Debug("Not configured to load secrets")
		// Intentionally do not fail. This is not required software to run. It needs to fail silent if it's not configured on an export.
		return nil
	}

	log.WithFields(log.Fields{
		"secureEnvironmentURL": secureEnvironmentURL,
	}).Debug("Attempting to load secure environment")

	if secureEnvironmentKey == "" {
		log.Debug("Cannot load secrets. No SECURE_ENVIRONMENT_KEY set")
		os.Exit(1)
		return nil
	}

	data, err := s3GetObject(secureEnvironmentURL)
	if err != nil {
		return err
	}

	log.Debug("Connecting to KMS")
	cipher, err := NewCipher()
	if err != nil {
		return nil
	}

	// Decrypt
	decryptedBytes, err := cipher.Decrypt(secureEnvironmentKey, data)
	if err != nil {
		return err
	}

	// Process file and export the variables
	decrypted := string(decryptedBytes)

	decryptedLines := strings.Split(decrypted, "\n")

	for lineNumber, line := range decryptedLines {
		line = strings.TrimSpace(line)
		if line == "" {
			log.Debugf("Empty line: %d", lineNumber)
			continue
		}
		if !envFileLineRegex.MatchString(line) {
			log.Debugf("Invalid line: %d: %s", lineNumber, line)
			continue
		}
		if line[0] == '#' {
			log.Debug("Comment line found")
			continue
		}
		splitLine := strings.Split(line, "=")
		key := splitLine[0]
		value := strings.Join(splitLine[1:], "=")

		fmt.Printf("export %s='%s'\n", key, escapeSingleQuote(value))
	}

	return nil
}
