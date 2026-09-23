package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type List struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
	Stats struct {
		Total     int `json:"total_items"`
		Completed int `json:"completed_items"`
	} `json:"stats"`
}

type Section struct {
	ID     int    `json:"id"`
	ListID int    `json:"list_id"`
	Name   string `json:"name"`
	Order  int    `json:"order"`
	Items  []Item `json:"items"`
}

type Item struct {
	ID          int    `json:"id"`
	SectionID   int    `json:"section_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Completed   bool   `json:"completed"`
	Uncertain   bool   `json:"uncertain"`
	Order       int    `json:"order"`
}

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func newClient(cfg config) *client {
	return &client{cfg.BaseURL + "/api/v1", cfg.Token, &http.Client{Timeout: 15 * time.Second}}
}

func (c *client) do(method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Message == "" {
			apiErr.Message = resp.Status
		}
		return fmt.Errorf("%s", apiErr.Message)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *client) lists() ([]List, error) {
	var out struct {
		Lists []List `json:"lists"`
	}
	err := c.do(http.MethodGet, "/lists", nil, &out)
	return out.Lists, err
}

func (c *client) sections(listID int) ([]Section, error) {
	var out struct {
		Sections []Section `json:"sections"`
	}
	err := c.do(http.MethodGet, fmt.Sprintf("/lists/%d/sections", listID), nil, &out)
	return out.Sections, err
}

func (c *client) createList(name string) error {
	return c.do(http.MethodPost, "/lists", map[string]any{"name": name, "icon": "cart"}, nil)
}

func (c *client) createSection(listID int, name string) error {
	return c.do(http.MethodPost, "/sections", map[string]any{"list_id": listID, "name": name}, nil)
}

func (c *client) createItem(sectionID int, name string, quantity int) error {
	return c.do(http.MethodPost, "/items", map[string]any{"section_id": sectionID, "name": name, "quantity": quantity}, nil)
}

func (c *client) updateList(id int, name string) error {
	return c.do(http.MethodPut, fmt.Sprintf("/lists/%d", id), map[string]any{"name": name}, nil)
}

func (c *client) updateSection(id int, name string) error {
	return c.do(http.MethodPut, fmt.Sprintf("/sections/%d", id), map[string]any{"name": name}, nil)
}

func (c *client) updateItem(id int, name string, quantity, sectionID, oldSectionID int) error {
	if err := c.do(http.MethodPut, fmt.Sprintf("/items/%d", id), map[string]any{"name": name, "quantity": quantity}, nil); err != nil {
		return err
	}
	if sectionID == oldSectionID {
		return nil
	}
	return c.do(http.MethodPost, fmt.Sprintf("/items/%d/move", id), map[string]any{"section_id": sectionID}, nil)
}

func (c *client) toggleItem(id int) error {
	return c.do(http.MethodPost, fmt.Sprintf("/items/%d/toggle", id), nil, nil)
}

func (c *client) delete(kind string, id int) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/%s/%d", kind, id), nil, nil)
}
