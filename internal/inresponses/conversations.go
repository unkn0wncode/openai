// Package inresponses / conversations.go contains the implementation of the ConversationCli interface.
package inresponses

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/responses"
)

// conversationCli performs API calls on a conversation object.
type conversationCli struct {
	*openai.Config
	data *responses.Conversation
}

// interface compliance checks
var _ responses.ConversationCli = (*conversationCli)(nil)

// BindConversationCli embeds the conversationCli in the conversation object.
func (c *Client) BindConversationCli(conv *responses.Conversation) {
	if conv == nil {
		return
	}

	conv.ConversationCli = conversationCli{
		Config: c.Config,
		data:   conv,
	}
}

// CreateConversation creates a new persistent conversation container.
func (c *Client) CreateConversation(metadata map[string]string, items ...any) (*responses.Conversation, error) {
	payload := struct {
		Metadata map[string]string `json:"metadata,omitempty"`
		Items    []any             `json:"items,omitempty"`
	}{
		Metadata: metadata,
	}
	if len(items) > 0 {
		payload.Items = items
	}

	body, err := openai.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conversation payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.BaseAPI+"v1/conversations", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation request: %w", err)
	}
	c.AddHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send conversation request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("conversation creation failed: %s %s", resp.Status, string(respBody))
	}

	var conv responses.Conversation
	if err := json.Unmarshal(respBody, &conv); err != nil {
		return nil, fmt.Errorf("failed to decode conversation response: %w", err)
	}

	c.BindConversationCli(&conv)
	return &conv, nil
}

// Conversation retrieves a conversation by ID.
func (c *Client) Conversation(conversationID string) (*responses.Conversation, error) {
	if conversationID == "" {
		return nil, errors.New("conversationID is empty")
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.BaseAPI+"v1/conversations/"+conversationID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation retrieve request: %w", err)
	}
	c.AddHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send conversation retrieve request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("conversation retrieve failed: %s %s", resp.Status, string(respBody))
	}

	var conv responses.Conversation
	if err := json.Unmarshal(respBody, &conv); err != nil {
		return nil, fmt.Errorf("failed to decode conversation: %w", err)
	}

	c.BindConversationCli(&conv)
	return &conv, nil
}

// Update sends the current metadata of the conversation to the API.
func (c conversationCli) Update() error {
	payload := struct {
		Metadata map[string]string `json:"metadata,omitempty"`
	}{
		Metadata: c.data.Metadata,
	}
	body, err := openai.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation update: %w", err)
	}

	resp, err := c.do(http.MethodPost, "", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(c.data); err != nil {
		return fmt.Errorf("failed to decode conversation update: %w", err)
	}

	return nil
}

// Delete removes the conversation from the API.
func (c conversationCli) Delete() error {
	resp, err := c.do(http.MethodDelete, "", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ListItems retrieves items stored in the conversation.
func (c conversationCli) ListItems(opts *responses.ConversationListOptions) (*responses.ConversationItemList, error) {
	if err := c.ensureReady(); err != nil {
		return nil, fmt.Errorf("conversationCli is not ready: %w", err)
	}

	endpoint, err := url.Parse(c.baseURL() + "/items")
	if err != nil {
		return nil, fmt.Errorf("failed to parse items endpoint: %w", err)
	}

	if opts != nil {
		values := endpoint.Query()
		if opts.Limit > 0 {
			values.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.FirstID != "" {
			values.Set("first_id", opts.FirstID)
		}
		if opts.LastID != "" {
			values.Set("last_id", opts.LastID)
		}
		addInclude(values, opts.Include)
		endpoint.RawQuery = values.Encode()
	}

	resp, err := c.doAbsolute(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list responses.ConversationItemList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode conversation items: %w", err)
	}

	err = list.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse conversation items: %w", err)
	}

	return &list, nil
}

// AppendItems adds new items to the conversation.
func (c conversationCli) AppendItems(include *responses.ConversationItemsInclude, items ...any) (*responses.ConversationItemList, error) {
	if len(items) == 0 {
		return nil, errors.New("responses: at least one item must be provided")
	}
	if err := c.ensureReady(); err != nil {
		return nil, fmt.Errorf("conversationCli is not ready: %w", err)
	}

	endpoint, err := url.Parse(c.baseURL() + "/items")
	if err != nil {
		return nil, fmt.Errorf("failed to parse append endpoint: %w", err)
	}
	values := endpoint.Query()
	addInclude(values, include)
	endpoint.RawQuery = values.Encode()

	payload := struct {
		Items []any `json:"items"`
	}{
		Items: items,
	}
	body, err := openai.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal append payload: %w", err)
	}

	resp, err := c.doAbsolute(http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list responses.ConversationItemList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode appended items: %w", err)
	}

	err = list.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse appended items: %w", err)
	}

	return &list, nil
}

// Item retrieves a single item from the conversation.
func (c conversationCli) Item(include *responses.ConversationItemsInclude, itemID string) (any, error) {
	if itemID == "" {
		return nil, errors.New("itemID is empty")
	}
	if err := c.ensureReady(); err != nil {
		return nil, fmt.Errorf("conversationCli is not ready: %w", err)
	}

	endpoint, err := url.Parse(c.baseURL() + "/items/" + itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse item endpoint: %w", err)
	}
	values := endpoint.Query()
	addInclude(values, include)
	endpoint.RawQuery = values.Encode()

	resp, err := c.doAbsolute(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var anyItem output.Any
	if err := json.NewDecoder(resp.Body).Decode(&anyItem); err != nil {
		return nil, fmt.Errorf("failed to decode conversation item: %w", err)
	}

	parsed, err := anyItem.Unmarshal()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation item: %w", err)
	}
	return parsed, nil
}

// DeleteItem removes a single item from the conversation.
func (c conversationCli) DeleteItem(itemID string) error {
	if itemID == "" {
		return errors.New("responses: itemID is required")
	}
	if err := c.ensureReady(); err != nil {
		return err
	}

	resp, err := c.do(http.MethodDelete, "/items/"+itemID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ensureReady checks if the conversationCli is ready to perform API calls:
// config is set and has an HTTP client, conversation is set and has an ID.
func (c conversationCli) ensureReady() error {
	switch {
	case c.Config == nil:
		return errors.New("responses: config is nil")
	case c.HTTPClient == nil:
		return errors.New("responses: http client is nil")
	case c.data == nil:
		return errors.New("responses: conversation is nil")
	case c.data.ID == "":
		return errors.New("responses: conversation ID is empty")
	default:
		return nil
	}
}

// baseURL builds a URL relevant to Config's BaseAPI address, including "conversations" and a
// conversation ID.
func (c conversationCli) baseURL() string {
	return c.BaseAPI + "v1/conversations/" + c.data.ID
}

// do performs an API call to the conversation's base URL with a given method and suffix.
func (c conversationCli) do(method, suffix string, body []byte) (*http.Response, error) {
	return c.doAbsolute(method, c.baseURL()+suffix, body)
}

// doAbsolute performs an API call to a given full URL with a given method and body.
func (c conversationCli) doAbsolute(method, fullURL string, body []byte) (*http.Response, error) {
	if err := c.ensureReady(); err != nil {
		return nil, fmt.Errorf("conversationCli is not ready: %w", err)
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("got status %s, but failed to read response body: %w", resp.Status, err)
		}
		return nil, fmt.Errorf("request failed with status %s, body: %s", resp.Status, string(data))
	}
	return resp, nil
}

// addInclude adds include flags to the URL query values.
func addInclude(values url.Values, include *responses.ConversationItemsInclude) {
	if include == nil {
		return
	}
	for _, flag := range include.Values() {
		if flag == "" {
			continue
		}
		values.Add("include[]", flag)
	}
}
