package s83

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Follow struct {
	publisher Publisher
	url       *url.URL
	handle    string
}

func NewFollow(key string, urlStr string, handle string) (Follow, error) {
	pub, err := NewPublisherFromKey(key)
	if err != nil {
		return Follow{}, err
	}

	url, err := url.Parse(urlStr)
	if err != nil {
		return Follow{}, err
	}

	return Follow{pub, url, handle}, nil
}

func (f Follow) String() string {
	return fmt.Sprintf("%s %s @ %s", f.publisher, f.handle, f.url.Host)
}

func (f Follow) GetBoard() (Board, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", f.url.String(), nil)
	if err != nil {
		return Board{}, err
	}

	// set headers
	req.Header.Set("Spring-Version", SpringVersion)
	// TODO: optional
	//req.Header.Set("If-Modified-Since", time.Now().UTC().Format(http.TimeFormat))

	// make request
	res, err := client.Do(req)
	if err != nil {
		return Board{}, err
	}

	if res.StatusCode != http.StatusOK {
		return Board{}, fmt.Errorf("Status code: %v", res.Status)
	}

	return NewBoardFromHTTP(f.publisher.String(), res.Header.Get("Spring-Signature"), res.Body)
}

// attempt to conform to the Springfile format of the demo client
func ParseSpringfileFollows(data []byte) []Follow {
	springfile := string(data)
	follows := []Follow{}

	handle := ""
	for _, line := range strings.Split(springfile, "\n") {
		line = strings.TrimSuffix(line, "\n")
		// skip blank/comment lines (and reset handle)
		if len(line) == 0 || line[0] == '#' {
			handle = ""
			continue
		}

		reSpringURL := regexp.MustCompile(`^http[s]?:\/\/(.*)\/([0-9A-Fa-f]{57}83e(0[1-9]|1[0-2])\d\d)$`)
		springURLMatch := reSpringURL.FindStringSubmatch(line)
		if springURLMatch != nil {
			key := springURLMatch[2]
			f, err := NewFollow(key, line, handle)
			if err != nil {
				continue
			}
			follows = append(follows, f)
			handle = ""
		} else {
			// keep line around as handle
			handle = line
		}
	}
	return follows
}
