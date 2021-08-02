package vault

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/invitae-ankit/ylp/util"
	"github.com/spf13/cobra"
)

type VaultResponse struct {
	Tokens string `json:"berossus_tokens"`
}

var (
	clientIds []string
	env       string
	config    VaultConfig
)

func New() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "vault",
		Short: "Vault related helper commands",
		Run: func(cmd *cobra.Command, args []string) {
			process()
		},
	}
	cmd.Flags().StringSliceVarP(&clientIds, "ids", "i", []string{}, "Client ids for which berossus token needs to be created. Ex. jane.doe,john.doe")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Vault environment name. Ex. dev prod")
	cmd.MarkFlagRequired("ids")

	return cmd
}

func process() {
	// Check if env is provided. If not then get it
	if env == "" {
		env = getEnvName()
	}
	config = NewVaultConfig(env)

	// get vault api client
	client := getClient()

	respData, err := getVaultData(client)
	util.HandleError(err)

	vaultData, _ := json.Marshal(respData)

	var currentData VaultResponse
	json.Unmarshal(vaultData, &currentData)
	currentTokens := currentData.Tokens

	// save current tokens to temp file
	tempFile, err := os.CreateTemp("", "berossus_tokens_old_*.txt")
	util.HandleError(err)
	util.SaveTofile(tempFile, currentTokens)
	fmt.Printf("Old tokens are saved in file: %s\n", tempFile.Name())

	//Create new tokens and append them to the existing token list
	tokenSlice := strings.Split(currentTokens, "\n")

	var newTokens []string
	for _, id := range clientIds {
		newTokenString := fmt.Sprintf(`\""%s"\" \""%s"\"`, id, uuid.New().String())
		newTokens = append(newTokens, newTokenString)
	}

	tokenSlice = append(tokenSlice[:len(tokenSlice)-1], newTokens...)
	tokenSlice = append(tokenSlice, "}")

	newFile, err := os.Create("berossus_tokens.txt")
	tokens := strings.Join(tokenSlice, "\n")
	util.HandleError(err)
	util.SaveTofile(newFile, tokens)
	fmt.Printf("New tokens are saved in file: %s\n", newFile.Name())

	if getConfirmationToUpload() {
		uploadNewTokens(client, tokens)
	}
}

func getClient() *api.Client {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	client, err := api.NewClient(&api.Config{Address: config.Addr, HttpClient: httpClient})
	util.HandleError(err)
	return client
}

func setTokenFromWeb(client *api.Client) {
	username, password := getCredentials()
	fmt.Println("Getting token from web")
	path := fmt.Sprintf("auth/ldap/login/%s", username)

	secret, err := client.Logical().Write(path, map[string]interface{}{
		"password": password,
	})
	util.HandleError(err)

	//set token
	client.SetToken(string(secret.Auth.ClientToken))
}

func getVaultData(client *api.Client) (map[string]interface{}, error) {
	setTokenFromWeb(client)
	resp, err := client.Logical().Read("secret/lims/berossus")
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func getEnvName() string {
	prompt := &survey.Select{
		Message: "Please select Environment?",
		Options: []string{"dev", "prod"},
	}

	var env string
	survey.AskOne(prompt, &env, survey.WithValidator(survey.Required))
	return env
}

func getCredentials() (string, string) {
	var qs = []*survey.Question{
		{
			Name:     "username",
			Prompt:   &survey.Input{Message: "Enter Username:"},
			Validate: survey.Required,
		},
		{
			Name:     "password",
			Prompt:   &survey.Password{Message: "Enter Password:"},
			Validate: survey.Required,
		},
	}

	answers := struct {
		Username string
		Password string
	}{}

	err := survey.Ask(qs, &answers)
	util.HandleError(err)
	return strings.TrimSpace(answers.Username), answers.Password
}

func getConfirmationToUpload() bool {
	msg := fmt.Sprintln("Do you want to upload new tokens to the following endpoints (y/n)?")
	for _, url := range config.VaultUrls {
		msg = msg + fmt.Sprintln("\t *", url)
	}

	prompt := &survey.Select{
		Message: msg,
		Options: []string{"yes", "no"},
	}

	var upload string
	survey.AskOne(prompt, &upload, survey.WithValidator(survey.Required))
	return upload == "yes"
}

func uploadNewTokens(client *api.Client, data string) {
	var wg sync.WaitGroup
	for _, url := range config.VaultUrls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			fmt.Println("Updating secret to:", url)
			_, err := client.Logical().Write(url, map[string]interface{}{
				"berossus_tokens": data,
			})
			util.HandleError(err)
		}(url)

		wg.Wait()
	}
}
