package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/databricks/bricks/retries"
	"github.com/pkg/browser"
)

// Bricks CLI GitHub OAuth App Client ID
const githubOauthClientID = "b91230382436c4592741"

func githubGetPAT(ctx context.Context) (string, error) {
	deviceRequest := url.Values{}
	deviceRequest.Set("client_id", githubOauthClientID)
	// TODO: scope
	response, err := http.PostForm("https://github.com/login/device/code", deviceRequest)
	if err != nil {
		return "", err
	}
	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	deviceResponse, err := url.ParseQuery(string(raw))
	if err != nil {
		return "", err
	}
	// TODO: give instructions to user and wait for the prompt
	userCode := deviceResponse.Get("user_code")
	err = clipboard.WriteAll(userCode)
	if err != nil {
		return "", fmt.Errorf("cannot copy to clipboard: %w", err)
	}
	verificationURL := deviceResponse.Get("verification_uri")
	fmt.Printf("\nEnter the following code on %s: \n\n%s\n\n(it should be in your clipboard)", verificationURL, userCode)
	err = browser.OpenURL(verificationURL)
	if err != nil {
		return "", fmt.Errorf("cannot open browser: %w", err)
	}
	var bearer string
	err = retries.Wait(ctx, 15*time.Minute, func() *retries.Err {
		form := url.Values{}
		form.Set("client_id", githubOauthClientID)
		form.Set("device_code", deviceResponse.Get("device_code"))
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		response, err := http.PostForm("https://github.com/login/oauth/access_token", form)
		if err != nil {
			return retries.Halt(err)
		}
		raw, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return retries.Continuef("failed to read body: %w", err)
		}
		result, err := url.ParseQuery(string(raw))
		if err != nil {
			return retries.Continuef("failed to parse body: %w", err)
		}
		bearer = result.Get("access_token")
		if bearer != "" {
			return nil
		}
		if result.Get("error") == "slow_down" {
			t, _ := strconv.Atoi(result.Get("interval"))
			time.Sleep(time.Duration(t) * time.Second)
			log.Printf("[WARN] Rate limited, sleeping for %d seconds", t)
		}
		reason := result.Get("error_description")
		if reason == "" {
			reason = "access token is not ready"
		}
		return retries.Continues(reason)
	})
	if err != nil {
		return "", fmt.Errorf("failed to acquire access token: %w", err)
	}
	raw, err = json.Marshal(struct {
		note   string
		scopes []string
	}{"test token", []string{}})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest("POST", "https://api.github.com/api/v3/authorizations", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	raw, _ = ioutil.ReadAll(response.Body)
	log.Printf("[INFO] %s", raw)
	// TODO: convert to PAT
	return bearer, nil
}
