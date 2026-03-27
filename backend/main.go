package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	graphqlDocID = "24368985919464652"
	igAppID      = "936619743392459"
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type Response struct {
	VideoURL string `json:"video_url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h(w, r)
	}
}

func extractShortcode(inputURL string) string {
	inputURL = strings.Split(inputURL, "?")[0]
	re := regexp.MustCompile(`instagram\.com/(?:[^/]+/)?(?:reel|p)/([^/?]+)`)
	m := re.FindStringSubmatch(inputURL)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractViaGraphQL(shortcode string) (string, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	pageURL := "https://www.instagram.com/reel/" + shortcode + "/"
	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	csrf := ""
	for _, c := range resp.Cookies() {
		if c.Name == "csrftoken" {
			csrf = c.Value
			break
		}
	}
	if csrf == "" {
		csrf = "missing"
	}


	variables := `{"shortcode":"` + shortcode + `"}`
	form := url.Values{}
	form.Set("variables", variables)
	form.Set("doc_id", graphqlDocID)
	body := form.Encode()

	postReq, _ := http.NewRequest("POST", "https://www.instagram.com/graphql/query", strings.NewReader(body))
	postReq.Header.Set("User-Agent", userAgent)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRFToken", csrf)
	postReq.Header.Set("X-IG-App-ID", igAppID)
	postReq.Header.Set("Referer", "https://www.instagram.com/")

	postResp, err := client.Do(postReq)
	if err != nil {
		return "", err
	}
	defer postResp.Body.Close()
	data, _ := io.ReadAll(postResp.Body)

	var result struct {
		Data struct {
			XdtAPI struct {
				Items []struct {
					VideoVersions []struct {
						URL string `json:"url"`
					} `json:"video_versions"`
				} `json:"items"`
			} `json:"xdt_api__v1__media__shortcode__web_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	items := result.Data.XdtAPI.Items
	if len(items) == 0 || len(items[0].VideoVersions) == 0 {
		return "", nil
	}
	return items[0].VideoVersions[0].URL, nil
}

func extractViaRegex(body []byte) string {
	patterns := []string{
		`"video_url":"([^"]+)"`,
		`"og:video" content="([^"]+)"`,
		`property="og:video" content="([^"]+)"`,
		`"url":"(https://[^"]+\.mp4[^"]*)"`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		m := re.FindStringSubmatch(string(body))
		if len(m) >= 2 && strings.HasPrefix(m[1], "http") {
			return strings.ReplaceAll(m[1], `\u0026`, "&")
		}
	}
	return ""
}

func extractViaYtDlp(reelURL string) (string, error) {
	cmd := exec.Command("yt-dlp", "--print", "url", reelURL)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func reelHandler(w http.ResponseWriter, r *http.Request) {
	inputURL := r.URL.Query().Get("url")
	if inputURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Missing URL"})
		return
	}

	shortcode := extractShortcode(inputURL)
	if shortcode == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid Instagram reel URL"})
		return
	}

	videoURL := ""


	if v, err := extractViaGraphQL(shortcode); err == nil && v != "" {
		videoURL = v
	}


	if videoURL == "" {
		req, _ := http.NewRequest("GET", inputURL, nil)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "*/*")
		resp, err := (&http.Client{}).Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			videoURL = extractViaRegex(body)
		}
	}


	if videoURL == "" {
		if v, err := extractViaYtDlp(inputURL); err == nil && v != "" {
			videoURL = v
		}
	}

	if videoURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Video not found. Reel may be private or Instagram format changed."})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{VideoURL: videoURL})
}

func main() {
	http.HandleFunc("/api/reel", cors(reelHandler))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	println("Server running on :" + port)
	http.ListenAndServe(":"+port, nil)
}