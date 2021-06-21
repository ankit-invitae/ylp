package vault

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/invitae-ankit/ylp/util"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	username string
	password string
	clientId []string
)

const vaultAddr = "https://vault.dev.locusdev.net/"

type Vault struct {
	Tokens string `json:"berossus_tokens"`
}

// vaultCmd represents the vault command
var Cmd = &cobra.Command{
	Use:   "vault",
	Short: "Vault related helper commands",
	Run: func(cmd *cobra.Command, args []string) {
		process()
	},
}

func init() {
	Cmd.Flags().StringSliceVarP(&clientId, "ids", "i", []string{}, "Client ids for which berossus token needs to be created. Ex. jane.doe,john.doe")
	Cmd.MarkFlagRequired("ids")
}

func process() {
	// get vault api client
	client := getClient()

	data, err := getTokenFromFile(client)
	if err != nil {
		fmt.Println("Token is not working, will try to generate new token")
		data = getTokenFromWeb(client)
	}
	respData, _ := json.Marshal(data)

	var currentData Vault
	json.Unmarshal(respData, &currentData)
	currentTokens := currentData.Tokens

	// save current tokens to temp file
	tempFile, err := os.CreateTemp("", "berossus_tokens_old_*.txt")
	util.HandleError(err)
	saveTofile(tempFile, currentTokens)
	fmt.Printf("Old tokens are saved in file: %s\n", tempFile.Name())

	//Create new tokens and append them to the existing token list
	tokenSlice := strings.Split(currentTokens, "\n")

	var newTokens []string
	for _, id := range clientId {
		newTokenString := fmt.Sprintf(`\""%s"\" \""%s"\"`, id, uuid.New().String())
		newTokens = append(newTokens, newTokenString)
	}

	tokenSlice = append(tokenSlice[:len(tokenSlice)-1], newTokens...)
	tokenSlice = append(tokenSlice, "}")

	newFile, err := os.Create("berossus_tokens.txt")
	tokens := strings.Join(tokenSlice, "\n")
	util.HandleError(err)
	saveTofile(newFile, tokens)
	fmt.Printf("New tokens are saved in file: %s\n", newFile.Name())

	if getConfirmationToUpload() {
		uploadNewTokens(client, tokens)
	}
}

func getClient() *api.Client {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	client, err := api.NewClient(&api.Config{Address: vaultAddr, HttpClient: httpClient})
	util.HandleError(err)
	return client
}

func getTokenFromFile(client *api.Client) (map[string]interface{}, error) {
	fmt.Println("Getting token from file")
	homeDir, err := os.UserHomeDir()
	util.HandleError(err)

	// read token from file. If its unable to read it from file then return err
	token, err := os.ReadFile(path.Join(homeDir, ".vault-token"))
	if err != nil {
		return nil, err
	}

	//set token
	client.SetToken(string(token))
	return getData(client)

}

func getTokenFromWeb(client *api.Client) map[string]interface{} {
	getCredentials()
	fmt.Println("Getting token from Web")
	path := fmt.Sprintf("auth/ldap/login/%s", username)

	secret, err := client.Logical().Write(path, map[string]interface{}{
		"password": password,
	})
	util.HandleError(err)

	client.SetToken(string(secret.Auth.ClientToken))

	resp, err := getData(client)
	util.HandleError(err)

	return resp
}

func getData(client *api.Client) (map[string]interface{}, error) {
	resp, err := client.Logical().Read("secret/lims/berossus")
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func saveTofile(file *os.File, data string) {
	file.WriteString(data)
	defer file.Close()
}

func getCredentials() {
	usernamePrompt := promptui.Prompt{
		Label: "username",
		Validate: func(input string) error {
			if len(strings.TrimSpace(input)) == 0 {
				return fmt.Errorf("username cannot be blank")
			}
			return nil
		},
	}

	var err error
	username, err = usernamePrompt.Run()
	util.HandleError(err)
	username = strings.TrimSpace(username)

	passwordPrompt := promptui.Prompt{
		Label: "password",
		Validate: func(input string) error {
			if len(strings.TrimSpace(input)) == 0 {
				return fmt.Errorf("password cannot be blank")
			}
			return nil
		},
		Mask: '*',
	}

	err = nil
	password, err = passwordPrompt.Run()
	util.HandleError(err)
	password = strings.TrimSpace(password)
}

func getConfirmationToUpload() bool {
	uploadPrompt := promptui.Prompt{
		Label: "Do you want to upload new tokens (y/n)?",
		Validate: func(input string) error {
			if input == "y" || input == "n" {
				return nil
			}
			return fmt.Errorf("please enter either y/n")
		},
	}

	value, err := uploadPrompt.Run()
	util.HandleError(err)
	return value == "y"
}

func uploadNewTokens(client *api.Client, data string) {
	urls := map[string][]string{
		"dev": {"secret/lims/berossus", "secret/lims/berossusTest"},
		"prd": {"secret/lims/berossus", "secret/lims/berossus2", "secret/lims/berossus-dlo", "secret/lims/berossus-pipe"},
	}

	var wg sync.WaitGroup
	for _, url := range urls["dev"] {
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
