package httpServices

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"passport-booking/types"
	"time"
)

type SSOClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *SSOClient {
	return &SSOClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *SSOClient) RequestRedirectToken(req ServiceUserRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/sso/service-user-request/", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", errors.New("SSO API returned non-OK status: " + resp.Status)
	}

	var apiResp ServiceUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}

	return apiResp.RedirectToken, nil
}

func (c *SSOClient) RequestLoginUser(req types.LoginRequest) (*types.LoginUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/sso/login-phone/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, errors.New("SSO Login API returned non-OK status: " + resp.Status)
	}

	var apiResp types.LoginUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func (c *SSOClient) RequestDMSLoginUser(req types.LoginDMSRequest) (*types.LoginUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/user/rms-user-land/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Restore the response body for JSON decoding
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, errors.New("SSO Login API returned non-OK status: " + resp.Status)
	}

	var apiResp types.LoginUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Print the decoded response structure
	return &apiResp, nil
}

func (c *SSOClient) RequestRegisterUser(req types.RegisterUserRequest) (*types.RegisterUserResponse, error) {
	body, err := json.Marshal(req)
	fmt.Printf("Request Register User: %s\n", string(body))
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/sso/register-service-user/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Only set Authorization header if Access token is provided and not empty
	if req.Access != "" {
		authHeader := "Bearer " + req.Access
		fmt.Printf("Setting Authorization header: '%s'\n", authHeader)
		httpReq.Header.Set("Authorization", authHeader)
	} else {
		fmt.Println("No Access token provided, making request without Authorization header")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, errors.New("SSO Register API returned non-OK status: " + resp.Status)
	}

	var apiResp types.RegisterUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}
