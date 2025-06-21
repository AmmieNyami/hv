package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"runtime"
	"time"
)

func exeDirectory() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	return path.Dir(exePath), nil
}

func isURLValid(str string) bool {
	parsedURL, err := url.Parse(str)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}

func errorToHttpError(w http.ResponseWriter, err error) {
	var caller string

	pc, file, line, ok := runtime.Caller(1)
	if ok {
		callerFunc := runtime.FuncForPC(pc)
		caller = fmt.Sprintf("%s:%d: %s:", file, line, callerFunc.Name())
	}

	log.Printf("%s %v\n", caller, err)

	var response Response

	var dbErr DatabaseError
	if errors.As(err, &dbErr) {
		response = Response{
			ErrorCode:   int(dbErr),
			ErrorString: dbErr.Error(),
			Data:        nil,
		}
	} else {
		response = Response{
			ErrorCode:   -1,
			ErrorString: "Internal server error",
			Data:        nil,
		}
	}

	WriteResponseHttp(response, http.StatusInternalServerError, w)
	return
}

func decodeJson(in io.Reader, out any, w http.ResponseWriter) (succeeded bool) {
	err := json.NewDecoder(in).Decode(out)
	if err != nil {
		WriteResponseHttp(Response{
			ErrorCode:   -1,
			ErrorString: "Invalid input",
			Data:        nil,
		}, http.StatusBadRequest, w)
		return false
	}

	return true
}

func getAuthData(r *http.Request) (username string, token string) {
	username = ""
	token = ""

	if usernameCookie, err := r.Cookie("username"); err == nil {
		username = usernameCookie.Value
	}

	if tokenCookie, err := r.Cookie("token"); err == nil {
		token = tokenCookie.Value
	}

	return
}

type RegisterUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func registerUser(db *Database, serverConfig ServerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if serverConfig.DisableRegistering {
			WriteResponseHttp(Response{
				ErrorCode:   -1,
				ErrorString: "User registering is disabled",
				Data:        nil,
			}, http.StatusUnauthorized, w)
			return
		}

		var newUser RegisterUserRequest
		if !decodeJson(r.Body, &newUser, w) {
			return
		}

		err := db.RegisterUser(newUser.Username, newUser.Password)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data:        nil,
		}, http.StatusOK, w)
	}
}

type LoginUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginUser(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var credentials LoginUserRequest
		if !decodeJson(r.Body, &credentials, w) {
			return
		}

		token, err := db.LoginUser(credentials.Username, credentials.Password)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    credentials.Username,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data:        nil,
		}, http.StatusOK, w)
	}
}

func logoutUser(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		err := db.LogoutUser(username, token)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data:        nil,
		}, http.StatusOK, w)
	}
}

type NeedsLoginResponse struct {
	NeedsLogin bool `json:"needs_login"`
}

func needsLogin(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		valid, err := db.IsAuthDataValid(username, token)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: NeedsLoginResponse{
				NeedsLogin: !valid,
			},
		}, http.StatusOK, w)
	}
}

type GetUsernameResponse struct {
	Username string `json:"username"`
}

func getUsername(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		actualUsername, err := db.GetUsername(username, token)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: GetUsernameResponse{
				Username: actualUsername,
			},
		}, http.StatusOK, w)
	}
}

type SearchDoujinsRequest struct {
	Query      string   `json:"query"`
	PageSize   int      `json:"page_size"`
	PageNumber int      `json:"page_number"`
	Tags       []string `json:"tags"`
	AntiTags   []string `json:"anti_tags"`
}

type SearchDoujinsResponse struct {
	Results SearchResult `json:"results"`
}

func searchDoujins(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var searchReq SearchDoujinsRequest
		if !decodeJson(r.Body, &searchReq, w) {
			return
		}

		results, err := db.SearchDoujins(
			username, token,
			searchReq.Query, searchReq.Tags, searchReq.AntiTags,
			searchReq.PageSize, searchReq.PageNumber,
		)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: SearchDoujinsResponse{
				Results: results,
			},
		}, http.StatusOK, w)
	}
}

type GetDoujinRequest struct {
	DoujinId int `json:"doujin_id"`
}

type GetDoujinResponse struct {
	Doujin Doujin `json:"doujin"`
}

func getDoujin(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var doujinReq GetDoujinRequest
		if !decodeJson(r.Body, &doujinReq, w) {
			return
		}

		doujin, err := db.GetDoujinMetadata(username, token, doujinReq.DoujinId)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: GetDoujinResponse{
				Doujin: doujin,
			},
		}, http.StatusOK, w)
	}
}

type GetPageRequest struct {
	PageId int `json:"page_id"`
}

func getPage(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var pageReq GetPageRequest
		if !decodeJson(r.Body, &pageReq, w) {
			return
		}

		imagePath, err := db.GetPageFilePath(username, token, pageReq.PageId)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		file, err := os.Open(imagePath)
		if err != nil {
			errorToHttpError(w, err)
			return
		}
		defer file.Close()

		var contentType string
		switch path.Ext(imagePath) {
		case ".gif":
			contentType = "image/gif"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".webp":
			contentType = "image/webp"
		default:
			contentType = "application/octet-stream"
		}

		h := w.Header()
		h.Del("Content-Length")
		h.Set("Content-Type", contentType)
		h.Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Failed to stream file `%s`: %v", imagePath, err)
		}
	}
}

type GetTagsResponse struct {
	Tags []string `json:"tags"`
}

func getTags(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		tags, err := db.GetAllTags(username, token)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: GetTagsResponse{
				Tags: tags,
			},
		}, http.StatusOK, w)
	}
}

type CreateTagSetRequest struct {
	Tags     []string `json:"tags"`
	AntiTags []string `json:"anti_tags"`
}

type CreateTagSetResponse struct {
	TagSetId int `json:"tag_set_id"`
}

func createTagSet(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var tagSetReq CreateTagSetRequest
		if !decodeJson(r.Body, &tagSetReq, w) {
			return
		}

		tagSetId, err := db.CreateTagSet(username, token, tagSetReq.Tags, tagSetReq.AntiTags)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: CreateTagSetResponse{
				TagSetId: tagSetId,
			},
		}, http.StatusOK, w)
	}
}

type DeleteTagSetRequest struct {
	TagSetId int `json:"tag_set_id"`
}

func deleteTagSet(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var tagSetReq DeleteTagSetRequest
		if !decodeJson(r.Body, &tagSetReq, w) {
			return
		}

		err := db.DeleteTagSet(username, token, tagSetReq.TagSetId)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data:        nil,
		}, http.StatusOK, w)
	}
}

type ChangeTagSetRequest struct {
	TagSetId int      `json:"tag_set_id"`
	Tags     []string `json:"tags"`
	AntiTags []string `json:"anti_tags"`
}

func changeTagSet(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		var tagSetReq ChangeTagSetRequest
		if !decodeJson(r.Body, &tagSetReq, w) {
			return
		}

		err := db.ChangeTagSet(username, token, tagSetReq.TagSetId, tagSetReq.Tags, tagSetReq.AntiTags)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data:        nil,
		}, http.StatusOK, w)
	}
}

type GetTagSetsResponse struct {
	TagSets []TagSet `json:"tag_sets"`
}

func getTagSets(db *Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, token := getAuthData(r)

		tagSets, err := db.GetTagSets(username, token)
		if err != nil {
			errorToHttpError(w, err)
			return
		}

		WriteResponseHttp(Response{
			ErrorCode:   0,
			ErrorString: "OK",
			Data: GetTagSetsResponse{
				TagSets: tagSets,
			},
		}, http.StatusOK, w)
	}
}

func unknownEndpointHandler(frontendURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL, err := url.Parse(frontendURL)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		r.URL.Host = targetURL.Host
		r.URL.Scheme = targetURL.Scheme
		r.Host = targetURL.Host

		proxy.ServeHTTP(w, r)
	}
}

type ServerConfig struct {
	FrontendURL  string `json:"frontend_url"`
	DatabasePath string `json:"database_path"`
	Port         int    `json:"port"`

	DisableRegistering bool `json:"disable_registering"`
}

func LoadServerConfig() ServerConfig {
	findFile := func(file string) string {
		exeDirectory, _ := exeDirectory()
		workDirectory, _ := os.Getwd()

		for _, directory := range []string{workDirectory, exeDirectory} {
			path := path.Join(directory, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}
			return path
		}

		fmt.Fprintf(os.Stderr, "ERROR: cound not find file `%s`\n", file)
		os.Exit(1)
		panic("unreachable")
	}

	// Load server config
	serverConfigFilePath := findFile("config.json")
	serverConfigFile, err := os.ReadFile(serverConfigFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open file `%s`: %v\n", serverConfigFilePath, err)
		os.Exit(1)
	}

	var serverConfig ServerConfig
	err = UnmarshalJsonWithComments(string(serverConfigFile), &serverConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to decode JSON file `%s`: %v\n", serverConfigFilePath, err)
		os.Exit(1)
	}

	// Validate fields
	if !isURLValid(serverConfig.FrontendURL) {
		fmt.Fprintf(os.Stderr, "ERROR: invalid frontend URL specified in configuration file\n")
		os.Exit(1)
	}

	databasePath := findFile(serverConfig.DatabasePath)

	if serverConfig.Port == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: invalid port specified in configuration file\n")
		os.Exit(1)
	}

	// Return validated struct
	return ServerConfig{
		FrontendURL:  serverConfig.FrontendURL,
		DatabasePath: databasePath,
		Port:         serverConfig.Port,

		DisableRegistering: serverConfig.DisableRegistering,
	}
}

func startServer(serverConfig ServerConfig) {
	db, err := NewDatabase(serverConfig.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Authentication
	http.HandleFunc("/api/v1/register", Method(registerUser(db, serverConfig), "POST"))
	http.HandleFunc("/api/v1/login", Method(loginUser(db), "POST"))
	http.HandleFunc("/api/v1/needsLogin", Method(needsLogin(db), "POST"))
	http.HandleFunc("/api/v1/logout", Method(logoutUser(db), "POST"))

	// Doujins
	http.HandleFunc("/api/v1/search", Method(searchDoujins(db), "POST"))
	http.HandleFunc("/api/v1/doujin", Method(getDoujin(db), "POST"))
	http.HandleFunc("/api/v1/page", Method(getPage(db), "POST"))

	// Tags
	http.HandleFunc("/api/v1/tags", Method(getTags(db), "POST"))
	http.HandleFunc("/api/v1/createTagSet", Method(createTagSet(db), "POST"))
	http.HandleFunc("/api/v1/deleteTagSet", Method(deleteTagSet(db), "POST"))
	http.HandleFunc("/api/v1/changeTagSet", Method(changeTagSet(db), "POST"))
	http.HandleFunc("/api/v1/getTagSets", Method(getTagSets(db), "POST"))

	http.HandleFunc("/api/v1/getUsername", Method(getUsername(db), "POST"))

	// Handle unknown endpoints
	http.HandleFunc("/", unknownEndpointHandler(serverConfig.FrontendURL))

	log.Printf("Server starting on port %d...\n", serverConfig.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverConfig.Port), nil))
}

func manageUsage(out io.Writer, programName string) {
	fmt.Fprintf(out, "USAGE: %s manage <COMMAND>\n", programName)
	fmt.Fprintf(out, "    COMMANDs:\n")
	fmt.Fprintf(out, "        import-doujin <FOLDER>               Imports the doujin in FOLDER to the database.\n")
	fmt.Fprintf(out, "                                             The folder must contain a metadata.json file\n")
	fmt.Fprintf(out, "                                             (see meta-format) and a sequence of image files named from 1\n")
	fmt.Fprintf(out, "                                             to N (including the extension), with each file being a page.\n")
	fmt.Fprintf(out, "                                             The numbers can be padded with zeroes.\n")
	fmt.Fprintf(out, "        import-doujins-from <FOLDER>         Imports all doujins in FOLDER to the database.\n")
	fmt.Fprintf(out, "                                             The folder must contain subfolders, each with a metadata.json\n")
	fmt.Fprintf(out, "                                             file (see meta-format) and a sequence image files named from 1\n")
	fmt.Fprintf(out, "                                             to N (including the extension), with each file being a page.\n")
	fmt.Fprintf(out, "                                             The numbers can be padded with zeroes.\n")
	fmt.Fprintf(out, "        register-user <USERNAME> <PASSWORD>  Registers a new user with username USERNAME and password\n")
	fmt.Fprintf(out, "                                             PASSWORD.\n")
	fmt.Fprintf(out, "        help                                 Prints this help.\n")
}

func usage(out io.Writer, programName string) {
	fmt.Fprintf(out, "USAGE: %s <SUBCOMMAND>\n", programName)
	fmt.Fprintf(out, "SUBCOMMANDs:\n")
	fmt.Fprintf(out, "    help              Prints this help.\n")
	fmt.Fprintf(out, "    meta-format       Prints help for the format of the metadata.json file used for\n")
	fmt.Fprintf(out, "                      importing doujins.\n")
	fmt.Fprintf(out, "    start             Starts the server.\n")
	fmt.Fprintf(out, "    manage <COMMAND>  Manage users and doujins. Provide `help` as a command to list\n")
	fmt.Fprintf(out, "                      the available commands.\n")
}

var commandLineArguments = append([]string(nil), os.Args...)

func popArg() string {
	if len(commandLineArguments) <= 0 {
		return ""
	}

	arg := commandLineArguments[0]
	commandLineArguments = commandLineArguments[1:]
	return arg
}

func main() {
	programName := popArg()

	subcommand := popArg()
	switch subcommand {
	case "start":
		startServer(LoadServerConfig())
		os.Exit(0)

	case "help":
		usage(os.Stdout, programName)
		os.Exit(0)

	case "meta-format":
		fmt.Println("The metadata.json File Format")
		fmt.Println("")
		fmt.Println("The metadata.json file must contain the following JSON structure, with none of the")
		fmt.Println("fields being optional:")
		fmt.Println("")
		fmt.Println("```")
		fmt.Println("{")
		fmt.Println("    \"title\": \"[AmmieNyami] Yume no Kyouka ~ Fantastical Ecstasy\",")
		fmt.Println("    \"subtitle\": \"[AmmieNyami] \\u5922\\u306E\\u72C2\\u83EF\\u3000\\u301C Fantastical Ecstasy\",")
		fmt.Println("    \"favorite_counts\": 69420,")
		fmt.Println("    \"upload_date\": \"1996-08-15T07:00:50-03:00\",")
		fmt.Println("    \"character\": [\"Amane Mitsuda\", \"Touma Hisui\"],")
		fmt.Println("    \"tag\": [\"yuri\", \"romance\", \"slice of life\"],")
		fmt.Println("    \"artist\": [\"AmmieNyami\"],")
		fmt.Println("    \"group\": [\"Team Scarlet Reverie\"],")
		fmt.Println("    \"language\": [\"english\"],")
		fmt.Println("    \"Pages\": 20")
		fmt.Println("}")
		fmt.Println("```")
		fmt.Println("")
		fmt.Println("Where:")
		fmt.Println("")
		fmt.Println("- `\"title\"` is the doujin's title;")
		fmt.Println("- `\"subtitle\"` is the doujin's subtitle;")
		fmt.Println("- `\"favorite_counts\"` is the rating the doujin received in the external website it")
		fmt.Println("  was downloaded from. It usually represents a number of views, likes, favorites, etc;")
		fmt.Println("- `\"upload_date\"` is either the date the doujin was uploaded to the external website")
		fmt.Println("  it was downloaded from or the date the doujin was first published or imported. The")
		fmt.Println("  date is in RFC 3339 format;")
		fmt.Println("- `\"character\"` is an array containing the doujin's main characters;")
		fmt.Println("- `\"tag\"` is an array containing the doujin's tags;")
		fmt.Println("- `\"artist\"` is an array containing the names of the artists that worked on the")
		fmt.Println("  doujin;")
		fmt.Println("- `\"group\"` is an array containing the names of the groups that worked on the doujin;")
		fmt.Println("- `\"language\"` is an array containing the languages used in the doujin;")
		fmt.Println("- `\"Pages\"` is the number of pages of the doujin.")
		os.Exit(0)

	case "manage":
		command := popArg()

		switch command {
		case "import-doujin":
			directory := popArg()
			if directory == "" {
				fmt.Fprintf(os.Stderr, "ERROR: no folder was provided for importing\n")
				manageUsage(os.Stderr, programName)
				os.Exit(1)
			}

			db, err := NewDatabase(LoadServerConfig().DatabasePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to create database: %v\n", err)
				os.Exit(1)
			}
			defer db.Close()

			err = db.RegisterDoujin(directory)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to register doujin in folder `%s`: %v\n", directory, err)
				os.Exit(1)
			}

			os.Exit(0)

		case "import-doujins-from":
			directory := popArg()
			if directory == "" {
				fmt.Fprintf(os.Stderr, "ERROR: no folder was provided for importing\n")
				manageUsage(os.Stderr, programName)
				os.Exit(1)
			}

			db, err := NewDatabase(LoadServerConfig().DatabasePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to create database: %v\n", err)
				os.Exit(1)
			}
			defer db.Close()

			entries, err := os.ReadDir(directory)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to read directory `%s`: %v\n", directory, err)
				os.Exit(1)
			}

			for _, e := range entries {
				if !e.IsDir() {
					continue
				}

				doujinPath := path.Join(directory, e.Name())
				err = db.RegisterDoujin(doujinPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: failed to register doujin in folder `%s`: %v\n", doujinPath, err)
					os.Exit(1)
				}
			}

			os.Exit(0)

		case "register-user":
			username := popArg()
			if username == "" {
				fmt.Fprintf(os.Stderr, "ERROR: no username was provided\n")
				manageUsage(os.Stderr, programName)
				os.Exit(1)
			}

			password := popArg()
			if password == "" {
				fmt.Fprintf(os.Stderr, "ERROR: no password was provided\n")
				manageUsage(os.Stderr, programName)
				os.Exit(1)
			}

			db, err := NewDatabase(LoadServerConfig().DatabasePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to create database: %v\n", err)
				os.Exit(1)
			}
			defer db.Close()

			err = db.RegisterUser(username, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: failed to register user: %v\n", err)
				os.Exit(1)
			}

			os.Exit(0)

		case "help":
			manageUsage(os.Stdout, programName)
			os.Exit(0)

		case "":
			fmt.Fprintf(os.Stderr, "ERROR: no command was provided to `manage`\n")
			manageUsage(os.Stderr, programName)
			os.Exit(1)

		default:
			fmt.Fprintf(os.Stderr, "ERROR: unknown command `%s`\n", command)
			manageUsage(os.Stderr, programName)
			os.Exit(1)
		}

	case "":
		fmt.Fprintf(os.Stderr, "ERROR: no subcommand was provided\n")
		usage(os.Stderr, programName)
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown subcommand `%s`\n", subcommand)
		usage(os.Stderr, programName)
		os.Exit(1)
	}
}
