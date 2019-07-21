package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/alecthomas/kingpin.v2"
)

var baseURL = "https://audioteka.com"

// App is a container wrapper for the http.Client,
// poviding methods for scraping
type App struct {
	Client *http.Client
}

// Item is a container for Name and Href of the items in the scrapped website
type Item struct {
	Name string
	Href string
}

func (app *App) getToken() string {
	loginURL := baseURL + "/cz/signin/login"
	client := app.Client

	response, err := client.Get(loginURL)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}

	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	token, _ := document.Find("input[name='_token']").Attr("value")

	return token
}

func (app *App) login(username string, password string) {
	client := app.Client

	authenticityToken := app.getToken()

	loginURL := baseURL + "/cz/user/login_check"

	data := url.Values{}
	data.Set("_token", authenticityToken)
	data.Set("_remember_me", "1")
	data.Set("login", "")
	data.Set("_failure_path", "login")
	data.Set("_username", username)
	data.Set("_password", password)
	fmt.Print(data.Encode())

	response, err := client.PostForm(loginURL, data)

	if err != nil {
		log.Fatalln(err)
	}

	defer response.Body.Close()

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}
}

func (app *App) logout() {
	client := app.Client

	logoutURL := baseURL + "/cz/user/logout"

	response, err := client.Get(logoutURL)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}

	defer response.Body.Close()
}

func (app *App) getItems() []Item {
	projectsURL := baseURL + "/cz/my-shelf"
	client := app.Client

	response, err := client.Get(projectsURL)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}

	defer response.Body.Close()

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	var items []Item

	document.Find(".shelf-item h3 a").Each(func(i int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Text())
		href, exist := s.Attr("href")
		if exist {
			item := Item{
				Name: name,
				Href: href,
			}
			items = append(items, item)
		}
	})

	return items
}

func (app *App) downloadLatest(itemURL string, outputFolder string) (string, error) {
	pathSegments := strings.Split(itemURL, "/")
	issue := pathSegments[len(pathSegments)-1]
	url := baseURL + "/cz/audiobook/" + issue + "/download"
	fmt.Print(url)
	client := app.Client

	response, err := client.Get(url)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}
	defer response.Body.Close()

	fpath := filepath.Join(outputFolder, issue+".zip")
	out, err := os.Create(fpath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, response.Body)
	return fpath, err
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func main() {
	var username = kingpin.Flag("username", "Set the username").Short('u').Required().String()
	var password = kingpin.Flag("password", "Set the password").Short('p').Required().String()
	var outputFolder = kingpin.Flag("output", "Set the ouput folder").Short('o').Default(".").String()
	var downloadItem = kingpin.Flag("item", "Optionally set the item to download").Short('i').String()
	kingpin.Parse()

	jar, _ := cookiejar.New(nil)

	app := App{
		Client: &http.Client{Jar: jar},
	}
	defer app.logout()

	app.login(*username, *password)

	var itemToDownload string
	if *downloadItem != "" {
		itemToDownload = *downloadItem
	} else {
		items := app.getItems()
		for index, item := range items {
			fmt.Printf("%d: %s %s\n", index+1, item.Name, item.Href)
		}
		itemToDownload = items[0].Href
	}

	filepath, err := app.downloadLatest(itemToDownload, *outputFolder)
	if err == nil {
		fmt.Printf("Successfully downloaded to file: %s\n", filepath)
	}
	dirName := strings.Split(filepath, ".")[0]
	fnames, err := unzip(filepath, dirName)
	if err == nil {
		os.Remove(filepath)
		fmt.Printf("%d files was successfuly extracted.\n", len(fnames))
	}
}
