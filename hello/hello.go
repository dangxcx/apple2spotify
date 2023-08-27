package hello

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/hbollon/go-edlib"
)

const (
	appleQueryByID   string = `https://itunes.apple.com/lookup?id=%s&country=us&entity=%s`
	spotifyQueryLink string = "https://api.spotify.com/v1/search?q=%s&type=%s"
)

var secrets struct {
	SpotifySecret string // apikey
	SpotifyClient string // personal access token
	// ...
}

func prettyPrintJSON(b []byte) {
	var out bytes.Buffer
	_ = json.Indent(&out, b, "", "  ")
	fmt.Println(string(out.Bytes()))
}

type appleResults struct {
	ArtistName     string `json:"artistName"`
	CollectionName string `json:"collectionName"`
	CollectionId   int    `json:"collectionId"`
}

type Link interface {
	ServiceQuery() string
	BaseUrl() string
	GetConvertedLink() (string, error)
}

type SpotifyConverter struct {
	Artist string
	Album  string
	Track  string
	Token  string
}

func (a SpotifyConverter) BaseUrl() string {
	return spotifyQueryLink
}

func (a SpotifyConverter) ServiceQuery() string {
	query := "artist:" + a.Artist + " " + "album:" + a.Album
	if a.Track != "" {
		query += " track:" + a.Track
	}
	return url.PathEscape(query)
}

func (a SpotifyConverter) GetConvertedLink() (string, error) {
	requestUrl := fmt.Sprintf(a.BaseUrl(), a.ServiceQuery(), "album")
	request, _ := http.NewRequest("GET", requestUrl, nil)
	fmt.Println(requestUrl)
	request.Header.Set("Authorization", "Bearer "+a.Token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("failed to make request to ", requestUrl)
		fmt.Println(err)
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	var results SpotifySearchResult
	err = json.Unmarshal(b, &results)
	if err != nil {
		return "", err
	}

	if len(results.Albums.Items) == 0 {
		return "", fmt.Errorf("no results")
	}
	sort.Slice(results.Albums.Items, func(i, j int) bool {
		return edlib.LevenshteinDistance(results.Albums.Items[i].Name, a.Album) < edlib.LevenshteinDistance(results.Albums.Items[j].Name, a.Album)
	})
	return results.Albums.Items[0].ExternalURL.Spotify, err
}

type AppleMusicConv struct {
	Artist string
	Album  string
	Track  string
}

func (a AppleMusicConv) ServiceQuery() string {
	return url.PathEscape(a.Artist + "+" + a.Album)
}

func (AppleMusicConv) BaseUrl() string {
	return "https://itunes.apple.com/search?term=%s&entity=%s"
}

func (a AppleMusicConv) GetConvertedLink() (string, error) {
	requestUrl := fmt.Sprintf(a.BaseUrl(), a.ServiceQuery(), "album")
	request, _ := http.NewRequest("GET", requestUrl, nil)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("failed to make request to ", requestUrl)
		fmt.Println(err)
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	type result struct {
		Results []appleResults `json:"results"`
	}

	var results result
	err = json.Unmarshal(b, &results)
	if len(results.Results) == 0 {
		return "", fmt.Errorf("no results")
	}
	sort.Slice(results.Results, func(i, j int) bool {
		return edlib.LevenshteinDistance(results.Results[i].CollectionName, a.Album) < edlib.LevenshteinDistance(results.Results[j].CollectionName, a.Album)
	})

	link := fmt.Sprintf("https://music.apple.com/us/album/%d", results.Results[0].CollectionId)
	return link, nil
}

type SpotifySearchResult struct {
	Albums SpotifyAlbumResult
}

type SpotifyAlbumResult struct {
	Items []SpotifyAlbumObj
}

type SpotifyAlbumObj struct {
	ExternalURL struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Name    string
	Artists []SpotifyArtist
}

type SpotifyArtist struct {
	Name string
}

func GetLinkType(link string, spotifyToken string) (Link, error) {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	fmt.Println("host is:", parsedLink.Hostname())
	switch parsedLink.Hostname() {
	case "music.apple.com":
		components := strings.Split(parsedLink.Path, "/")
		fmt.Println(components)
		var entity string
		switch components[2] {
		case "album":
			entity = "album"
		case "artist":
			entity = "musicArtist"
		case "song":
			entity = "musicTrack"
		default:
			entity = "music"
		}
		conv, err := getAppleLinkInfo(components[len(components)-1], entity)
		if err != nil {
			return SpotifyConverter{}, err
		}
		conv.Token = spotifyToken
		return conv, nil
	case "open.spotify.com":
		fmt.Println(parsedLink.Path)
		return getSpotifyAlbumInfo(parsedLink.Path, spotifyToken)
	}
	return SpotifyConverter{}, fmt.Errorf("unknown service")
}

type ConvertLinkRequest struct {
	ObjectURL string
}

type ConvertLinkResponse struct {
	Link string
}

//encore:api public method=POST path=/convert
func ConvertLink(ctx context.Context, req *ConvertLinkRequest) (*ConvertLinkResponse, error) {
	spotifyToken, err := getSpotifyAuthToken()
	if err != nil {
		return nil, err
	}
	conv, err := GetLinkType(req.ObjectURL, spotifyToken)
	if err != nil {
		return nil, err
	}

	link, err := conv.GetConvertedLink()
	if err != nil {
		return nil, err
	}

	return &ConvertLinkResponse{
		Link: link,
	}, nil
}

func getSpotifyAlbumInfo(albumPath, token string) (AppleMusicConv, error) {
	requestUrl := "https://api.spotify.com/v1" + albumPath + "?market=US"
	requestUrl = strings.Replace(requestUrl, "album", "albums", 1)
	request, _ := http.NewRequest("GET", requestUrl, nil)
	fmt.Println(requestUrl)
	request.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("failed to make request to ", requestUrl)
		fmt.Println(err)
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var albumResult SpotifyAlbumObj
	err = json.Unmarshal(b, &albumResult)
	if err != nil {
		return AppleMusicConv{}, nil
	}

	return AppleMusicConv{
		Artist: albumResult.Artists[0].Name,
		Album:  albumResult.Name,
	}, nil
}

func getAppleLinkInfo(id string, entityType string) (*SpotifyConverter, error) {
	getQuery := fmt.Sprintf(appleQueryByID, id, entityType)
	fmt.Println(getQuery)
	resp, err := http.Get(getQuery)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	type result struct {
		Results []appleResults `json:"results"`
	}

	var resultsList result
	err = json.Unmarshal(b, &resultsList)
	if err != nil {
		return nil, err
	}

	conv := SpotifyConverter{
		Artist: resultsList.Results[0].ArtistName,
		Album:  resultsList.Results[0].CollectionName,
	}

	return &conv, nil
}

func getSpotifyAuthToken() (string, error) {
	authURL := "https://accounts.spotify.com/api/token"

	// Create a new HTTP client
	client := &http.Client{}

	// Prepare the request body
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	req, err := http.NewRequest("POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", err
	}

	// Set the basic authentication header
	req.SetBasicAuth(secrets.SpotifyClient, secrets.SpotifySecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Request failed with status code:", resp.StatusCode)
		return "", err
	}

	// Read the response body
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding response:", err)
		return "", err
	}

	// Extract the token from the response
	accessToken := result["access_token"].(string)
	fmt.Println("Access Token:", accessToken)
	return accessToken, err
}
