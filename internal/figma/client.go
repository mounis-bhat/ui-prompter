package figma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	APIKey string
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

// ExtractFileKeyAndNodeID parses a figma URL
// e.g. https://www.figma.com/file/abc123xyz/My-File?node-id=1-2
// or https://www.figma.com/design/abc123xyz/My-File?node-id=1-2
func ExtractFileKeyAndNodeID(figmaURL string) (string, string, error) {
	u, err := url.Parse(figmaURL)
	if err != nil {
		return "", "", err
	}

	pathParts := strings.Split(u.Path, "/")
	var fileKey string
	for i, part := range pathParts {
		if (part == "file" || part == "design" || part == "board") && i+1 < len(pathParts) {
			fileKey = pathParts[i+1]
			break
		}
	}

	if fileKey == "" {
		return "", "", fmt.Errorf("invalid figma URL: could not find file key")
	}

	nodeID := u.Query().Get("node-id")
	if nodeID == "" {
		return "", "", fmt.Errorf("invalid figma URL: could not find node-id query parameter")
	}
	
	// node-id can be format "1:2" or "1-2", API requires "1:2" usually, but sometimes accepts both.
	// We'll replace dash with colon just in case.
	nodeID = strings.ReplaceAll(nodeID, "-", ":")

	return fileKey, nodeID, nil
}

func (c *Client) GetNode(fileKey, nodeID string) (*Node, error) {
	apiURL := fmt.Sprintf("https://api.figma.com/v1/files/%s/nodes?ids=%s", fileKey, url.QueryEscape(nodeID))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Figma-Token", c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("figma API error: status %d", resp.StatusCode)
	}

	var result FileNodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// nodeID in response keys might preserve the original requested string
	nodeContent, ok := result.Nodes[nodeID]
	if !ok {
		// Try replacing : with - in case response uses different format
		altNodeID := strings.ReplaceAll(nodeID, ":", "-")
		nodeContent, ok = result.Nodes[altNodeID]
		if !ok {
			return nil, fmt.Errorf("node %s not found in response", nodeID)
		}
	}

	return &nodeContent.Document, nil
}
