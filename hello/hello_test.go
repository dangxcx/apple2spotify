package hello

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWorld(t *testing.T) {
	t.Parallel()
	const in = "Jane Doe"
	resp, err := World(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Message; !strings.Contains(got, in) {
		t.Errorf("World(%q) = %q, expected to contain %q", in, got, in)
	}
}

type tCase struct {
	Raw string
}

func TestGetLink(t *testing.T) {
	t.Parallel()
	cases := []tCase{
		{"https://open.spotify.com/album/2VYo0PSqdxVTMI0ydKUtoL"},
		//{"https://music.apple.com/us/artist/the-armed/580347042"},
	}
	for _, tc := range cases {
		token, err := getSpotifyAuthToken()
		if err != nil {
			fmt.Println(err)
		}
		conv, _ := GetLinkType(tc.Raw, token)
		requestUrl := fmt.Sprintf(conv.BaseUrl(), conv.ServiceQuery(), "album")

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
		prettyPrintJSON(b)
	}
}
