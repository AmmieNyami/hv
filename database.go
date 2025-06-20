package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/argon2"
)

const (
	PasswordSaltLength = 16
	SessionTokenLength = 30
	MaxTokensPerUser   = 20
)

func jsonEncode(v any) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

func base64encode(bytes []byte) string {
	return base64.URLEncoding.EncodeToString(bytes)
}

func randomString(n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate secure random bytes: %v", err))
	}

	return base64encode(bytes)
}

func hashPassword(password string, salt string) string {
	const iterations = 3
	const memory = 65536
	const parallelism = 4
	const keyLen = 32
	return base64encode(argon2.IDKey([]byte(password), []byte(salt), iterations, memory, parallelism, keyLen))
}

func hashToken(token string, salt string) string {
	bytes := sha256.Sum256([]byte(token + salt))
	return base64encode(bytes[:])
}

func sqlExec(db *sql.DB, query string, args ...any) (sql.Result, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("`%s`: %v", query, err)
	}
	return res, nil
}

func removeExtension(fileName string) string {
	if ext := path.Ext(fileName); ext != "" {
		return fileName[:len(fileName)-len(ext)]
	}
	return fileName
}

func pageNameToPageNumber(fileName string) int {
	noExtName := removeExtension(fileName)

	pageNumber, err := strconv.ParseUint(noExtName, 10, 32)
	if err != nil {
		return -1
	}

	return int(pageNumber)
}

func isUsernameValid(username string) bool {
	if len(username) < 1 {
		return false
	}

	allowedCharacters := []rune{
		'A', 'a', 'B', 'b', 'C', 'c', 'D', 'd', 'E', 'e',
		'F', 'f', 'G', 'g', 'H', 'h', 'I', 'i', 'J', 'j',
		'K', 'k', 'L', 'l', 'M', 'm', 'N', 'n', 'O', 'o',
		'P', 'p', 'Q', 'q', 'R', 'r', 'S', 's', 'T', 't',
		'U', 'u', 'V', 'v', 'W', 'w', 'X', 'x', 'Y', 'y',
		'Z', 'z', '0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9', '.', '_', '-',
	}

	for _, c := range username {
		if !slices.Contains(allowedCharacters, c) {
			return false
		}
	}

	return true
}

func isPasswordValid(password string) bool {
	return len(password) > 0
}

type SessionTokens string

var (
	SessionTokensErrorInvalidSlice  = errors.New("Invalid session tokens slice")
	SessionTokensErrorInvalidString = errors.New("Invalid session tokens string")
	SessionTokensErrorUnknownToken  = errors.New("Unknown token")
)

func NewSessionTokens(slice [][]string) (SessionTokens, error) {
	for _, x := range slice {
		if len(x) != 2 {
			return SessionTokens(""), SessionTokensErrorInvalidSlice
		}
	}

	return SessionTokens(jsonEncode(slice)), nil
}

func (tokens SessionTokens) ToSlice() ([][]string, error) {
	if tokens == SessionTokens("") {
		return [][]string{}, nil
	}

	tokensSlice := [][]string{}
	err := json.Unmarshal([]byte(tokens), &tokensSlice)
	if err != nil {
		return nil, SessionTokensErrorInvalidString
	}
	return tokensSlice, nil
}

func (tokens *SessionTokens) AppendNew() (string, error) {
	tokensSlice, err := tokens.ToSlice()
	if err != nil {
		return "", err
	}

	token := randomString(SessionTokenLength)

	salt := randomString(PasswordSaltLength)
	hash := hashToken(token, salt)

	tokensSlice = append(tokensSlice, []string{hash, salt})
	if len(tokensSlice) >= MaxTokensPerUser {
		tokensSlice = tokensSlice[1:]
	}

	newTokens, err := NewSessionTokens(tokensSlice)
	if err != nil {
		return "", err
	}

	*tokens = newTokens

	return token, nil
}

func (tokens *SessionTokens) RemoveToken(token string) error {
	tokensSlice, err := tokens.ToSlice()
	if err != nil {
		return err
	}

	tokenIndex := -1
	for i, t := range tokensSlice {
		if hashToken(token, t[1]) != t[0] {
			continue
		}

		tokenIndex = i
		break
	}

	if tokenIndex == -1 {
		return SessionTokensErrorUnknownToken
	}

	tokensSlice = slices.Delete(tokensSlice, tokenIndex, tokenIndex+1)

	newTokens, err := NewSessionTokens(tokensSlice)
	if err != nil {
		return err
	}

	*tokens = newTokens

	return nil
}

func (tokens SessionTokens) HasToken(token string) (bool, error) {
	tokensSlice, err := tokens.ToSlice()
	if err != nil {
		return false, err
	}

	for _, t := range tokensSlice {
		if hashToken(token, t[1]) == t[0] {
			return true, nil
		}
	}

	return false, nil
}

type DatabaseError int

const (
	DatabaseErrorInvalidMetadata DatabaseError = iota + 1
	DatabaseErrorInvalidSchema
	DatabaseErrorExistentUser
	DatabaseErrorInexistentUser
	DatabaseErrorInvalidPassword
	DatabaseErrorDisallowedUsername
	DatabaseErrorDisallowedPassword
	DatabaseErrorInvalidToken
	DatabaseErrorInvalidPageNumber
	DatabaseErrorInvalidId
	DatabaseErrorUnauthorized
	DatabaseErrorInvalidPageSize

	DatabaseErrorCount
)

var databaseErrorMessages = []string{
	DatabaseErrorInvalidMetadata:    "Invalid database metadata",
	DatabaseErrorInvalidSchema:      "Invalid database schema",
	DatabaseErrorExistentUser:       "User already exists in database",
	DatabaseErrorInexistentUser:     "User does not exist in database",
	DatabaseErrorInvalidPassword:    "Invalid password",
	DatabaseErrorDisallowedUsername: "Disallowed username",
	DatabaseErrorDisallowedPassword: "Disallowed password",
	DatabaseErrorInvalidToken:       "Invalid token",
	DatabaseErrorInvalidPageNumber:  "Invalid page number",
	DatabaseErrorInvalidId:          "Invalid ID",
	DatabaseErrorUnauthorized:       "Unauthorized",
	DatabaseErrorInvalidPageSize:    "Invalid page size",
}

func init() {
	if int(DatabaseErrorCount) != len(databaseErrorMessages) {
		panic("Not all error codes have databaseErrorMessages equivalents")
	}
}

func (err DatabaseError) Error() string {
	if int(err) >= len(databaseErrorMessages) {
		return fmt.Sprintf("Unknown error code %d", err)
	}
	return databaseErrorMessages[err]
}

type DoujinImportMetadata struct {
	Title          string    `json:"title"`
	Subtitle       string    `json:"subtitle"`
	ExternalRating int       `json:"favorite_counts"`
	UploadDate     time.Time `json:"upload_date"`
	Characters     []string  `json:"character"`
	Tags           []string  `json:"tag"`
	Artists        []string  `json:"artist"`
	Groups         []string  `json:"group"`
	Languages      []string  `json:"language"`
	Pages          int       `json:"pages"`
}

type Doujin struct {
	Id             int      `json:"id"`
	Title          string   `json:"title"`
	Subtitle       string   `json:"subtitle"`
	UploadDate     string   `json:"upload_date"`
	ExternalRating int      `json:"external_rating"`
	Tags           []string `json:"tags"`
	Characters     []string `json:"characters"`
	Artists        []string `json:"artists"`
	Groups         []string `json:"groups"`
	Languages      []string `json:"languages"`
	Pages          [][]int  `json:"pages"`
}

type SearchResult struct {
	Entries    []Doujin `json:"entries"`
	TotalPages int      `json:"total_pages"`
}

type TagSet struct {
	Id       int      `json:"id"`
	Tags     []string `json:"tags"`
	AntiTags []string `json:"anti_tags"`
}

type Database struct {
	db *sql.DB
}

func NewDatabase(databasePath string) (*Database, error) {
	isDatabaseInitialized := func(db *sql.DB) (string, error) {
		var appName string
		var schemaVersion string
		err := db.QueryRow(`SELECT app_name, schema_version FROM "META"`).Scan(&appName, &schemaVersion)
		if err != nil {
			return "", nil
		}

		if appName != "hv" {
			return "", DatabaseErrorInvalidMetadata
		}

		return schemaVersion, nil
	}

	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatal(err)
	}

	errored := true
	defer func() {
		if errored {
			db.Close()
		}
	}()

	schemaVersion, err := isDatabaseInitialized(db)
	if err != nil {
		return nil, err
	}

	_, err = sqlExec(db, "PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	if schemaVersion == "" {
		_, err = sqlExec(db, `CREATE TABLE "META" (app_name TEXT NOT NULL, schema_version TEXT NOT NULL);
							  INSERT INTO "META" (app_name, schema_version) VALUES ("hv", "v1")`)
		if err != nil {
			return nil, err
		}

		_, err = sqlExec(db, `CREATE TABLE Users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			password_salt TEXT NOT NULL,
			session_tokens TEXT NOT NULL
		)`)
		if err != nil {
			return nil, err
		}

		_, err = sqlExec(db, `CREATE TABLE Doujins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			subtitle TEXT NOT NULL,
			upload_date TEXT NOT NULL,
			external_rating INTEGER NOT NULL,
			tags TEXT NOT NULL,
			characters TEXT NOT NULL,
			artists TEXT NOT NULL,
			groups TEXT NOT NULL,
			languages TEXT NOT NULL,
			pages INTEGER NOT NULL
		)`)
		if err != nil {
			return nil, err
		}

		_, err = sqlExec(db, `CREATE TABLE DoujinPages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doujin_id INTEGER NOT NULL,
			page_path TEXT NOT NULL,
			page_number INTEGER NOT NULL,

			FOREIGN KEY (doujin_id) REFERENCES Doujins(id) ON DELETE CASCADE
		)`)
		if err != nil {
			return nil, err
		}

		_, err = sqlExec(db, `CREATE TABLE TagSets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			tags TEXT NOT NULL,
			anti_tags TEXT NOT NULL,

			FOREIGN KEY (user_id) REFERENCES Users(id) ON DELETE CASCADE
		)`)
		if err != nil {
			return nil, err
		}
	} else if schemaVersion != "v1" {
		return nil, DatabaseErrorInvalidSchema
	}

	errored = false
	return &Database{db}, nil
}

func (db *Database) authenticateUser(username string, token string) (int, error) {
	var userId int
	var sessionTokens SessionTokens
	err := db.db.QueryRow(
		`SELECT id, session_tokens FROM Users WHERE username = ? COLLATE NOCASE`,
		username,
	).Scan(&userId, &sessionTokens)

	if err == sql.ErrNoRows {
		return 0, DatabaseErrorInexistentUser
	}

	if err != nil {
		return 0, err
	}

	hasToken, err := sessionTokens.HasToken(token)
	if err != nil {
		return 0, err
	}

	if !hasToken {
		return 0, DatabaseErrorInvalidToken
	}

	return userId, nil
}

func (db *Database) RegisterDoujin(folderPath string) error {
	filePath := path.Join(folderPath, "metadata.json")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open file `%s`: %w", filePath, err)
	}
	defer file.Close()

	var doujinMeta DoujinImportMetadata
	err = json.NewDecoder(file).Decode(&doujinMeta)
	if err != nil {
		return fmt.Errorf("Failed to decode JSON file `%s`: %w", filePath, err)
	}

	tags := jsonEncode(doujinMeta.Tags)
	characters := jsonEncode(doujinMeta.Characters)
	artists := jsonEncode(doujinMeta.Artists)
	groups := jsonEncode(doujinMeta.Groups)
	languages := jsonEncode(doujinMeta.Languages)
	uploadDate := doujinMeta.UploadDate.Format(time.RFC3339)

	result, err := sqlExec(
		db.db,
		`INSERT INTO Doujins (title, subtitle, upload_date, external_rating, tags, characters, artists, groups, languages, pages)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doujinMeta.Title, doujinMeta.Subtitle, uploadDate, doujinMeta.ExternalRating,
		tags, characters, artists, groups, languages, doujinMeta.Pages,
	)
	if err != nil {
		return err
	}

	doujinId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("Failed to read directory `%s`: %w", folderPath, err)
	}

	log.Printf("Importing doujin in folder `%s`\n", folderPath)

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	importedPages := []int{}
	for _, e := range entries {
		pageNumber := pageNameToPageNumber(e.Name())
		if e.IsDir() || pageNumber == -1 {
			continue
		}

		_, err = tx.Exec(
			"INSERT INTO DoujinPages (doujin_id, page_path, page_number) VALUES (?, ?, ?)",
			doujinId, path.Join(folderPath, e.Name()), pageNumber,
		)
		if err != nil {
			return err
		}

		importedPages = append(importedPages, pageNumber)
	}

	if len(importedPages) < doujinMeta.Pages {
		return fmt.Errorf("Some pages are missing in the folder (found %d out of %d)", len(importedPages), doujinMeta.Pages)
	}

	if len(importedPages) > doujinMeta.Pages {
		return fmt.Errorf("The folder contains to many pages (found %d out of %d)", len(importedPages), doujinMeta.Pages)
	}

	sort.Ints(importedPages)
	for i, pageNumber := range importedPages {
		if i+1 != pageNumber {
			return fmt.Errorf("The pages in the folder are not sequential (expected page %d, found %d)", i+1, pageNumber)
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) RegisterUser(username string, password string) error {
	if !isUsernameValid(username) {
		return DatabaseErrorDisallowedUsername
	}

	if !isPasswordValid(password) {
		return DatabaseErrorDisallowedPassword
	}

	err := db.db.QueryRow(`SELECT 1 FROM Users WHERE username = ? COLLATE NOCASE`, username).Scan(new(int))
	if err == nil {
		return DatabaseErrorExistentUser
	}

	if err != sql.ErrNoRows {
		return err
	}

	passwordSalt := randomString(PasswordSaltLength)
	passwordHash := hashPassword(password, passwordSalt)

	_, err = sqlExec(
		db.db,
		"INSERT INTO Users (username, password_hash, password_salt, session_tokens) VALUES (?, ?, ?, ?)",
		username, passwordHash, passwordSalt, "",
	)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) LoginUser(username string, password string) (string, error) {
	var passwordHash string
	var passwordSalt string
	var sessionTokens SessionTokens
	err := db.db.QueryRow(
		`SELECT password_hash, password_salt, session_tokens FROM Users WHERE username = ? COLLATE NOCASE`,
		username,
	).Scan(&passwordHash, &passwordSalt, &sessionTokens)

	if err == sql.ErrNoRows {
		return "", DatabaseErrorInexistentUser
	}

	if err != nil {
		return "", err
	}

	if hashPassword(password, passwordSalt) != passwordHash {
		return "", DatabaseErrorInvalidPassword
	}

	token, err := sessionTokens.AppendNew()
	if err != nil {
		return "", err
	}

	_, err = sqlExec(
		db.db,
		"UPDATE Users SET session_tokens = ? WHERE username = ? COLLATE NOCASE",
		sessionTokens, username,
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

// This function is needed because even though it requires the username as a
// parameter, the username parameter doesn't need to be properly capitalized.
// As such, this function will always return the username with proper
// capitalization.
func (db *Database) GetUsername(username string, token string) (string, error) {
	userId, err := db.authenticateUser(username, token)
	if err != nil {
		return "", err
	}

	var actualUsername string
	err = db.db.QueryRow(`SELECT username FROM Users WHERE id = ?`, userId).Scan(&actualUsername)
	if err != nil {
		return "", err
	}

	return actualUsername, nil
}

func escapeSqlLike(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(s)
}

func buildSearchQuery(
	query string,
	tags []string,
	antiTags []string,
	pageSize int,
	pageNumber int,
	count bool,
) (string, []any) {
	var (
		queryBuilder    strings.Builder
		queryParameters []any
		sqlLikeQuery    = "%" + escapeSqlLike(query) + "%"
	)

	// Select
	if count {
		queryBuilder.WriteString("SELECT COUNT(*)")
	} else {
		queryBuilder.WriteString(
			"SELECT id, title, subtitle, upload_date, external_rating, tags, characters, artists, groups, languages")
	}

	// Basic search
	queryBuilder.WriteString(`
		FROM Doujins
		WHERE (title LIKE ? ESCAPE '\' OR subtitle LIKE ? ESCAPE '\')
	`)
	queryParameters = append(queryParameters, sqlLikeQuery, sqlLikeQuery)

	// Tags
	for _, tag := range tags {
		queryBuilder.WriteString(`
			AND EXISTS (
				SELECT 1 FROM json_each(Doujins.tags) AS jt
				WHERE jt.value = ?
			)
		`)
		queryParameters = append(queryParameters, tag)
	}

	// Anti-tags
	for _, tag := range antiTags {
		queryBuilder.WriteString(`
			AND NOT EXISTS (
				SELECT 1 FROM json_each(Doujins.tags) AS jt
				WHERE jt.value = ?
			)
		`)
		queryParameters = append(queryParameters, tag)
	}

	// Pagination
	if !count {
		queryBuilder.WriteString(`
			ORDER BY upload_date DESC
			LIMIT ? OFFSET ?
		`)
		queryParameters = append(queryParameters, pageSize, pageSize*(pageNumber-1))
	}

	return queryBuilder.String(), queryParameters
}

func (db *Database) SearchDoujins(
	username string, token string,
	query string, tags []string, antiTags []string, pageSize int, pageNumber int,
) (SearchResult, error) {
	_, err := db.authenticateUser(username, token)
	if err != nil {
		return SearchResult{}, err
	}

	if pageSize < 1 || pageSize > 100 {
		return SearchResult{}, DatabaseErrorInvalidPageSize
	}

	if pageNumber < 1 {
		return SearchResult{}, DatabaseErrorInvalidPageNumber
	}

	// Count query
	var resultsCount int
	countQuery, countQueryParameters := buildSearchQuery(query, tags, antiTags, pageSize, pageNumber, true)
	err = db.db.QueryRow(countQuery, countQueryParameters...).Scan(&resultsCount)
	if err != nil {
		return SearchResult{}, err
	}

	if resultsCount < 1 {
		return SearchResult{}, nil
	}
	totalPages := int(math.Ceil(float64(resultsCount) / float64(pageSize)))

	if pageNumber > totalPages {
		return SearchResult{}, DatabaseErrorInvalidPageNumber
	}

	// Search query
	searchQuery, searchQueryParameters := buildSearchQuery(query, tags, antiTags, pageSize, pageNumber, false)
	rows, err := db.db.Query(searchQuery, searchQueryParameters...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()

	doujins := []Doujin{}
	for rows.Next() {
		var id int
		var title string
		var subtitle string
		var uploadDate string
		var externalRating int
		var tagsJson string
		var charactersJson string
		var artistsJson string
		var groupsJson string
		var languagesJson string

		err = rows.Scan(
			&id,
			&title,
			&subtitle,
			&uploadDate,
			&externalRating,
			&tagsJson,
			&charactersJson,
			&artistsJson,
			&groupsJson,
			&languagesJson,
		)
		if err != nil {
			return SearchResult{}, err
		}

		var tags []string
		err = json.Unmarshal([]byte(tagsJson), &tags)
		if err != nil {
			return SearchResult{}, fmt.Errorf("Got invalid JSON from database")
		}

		var characters []string
		err = json.Unmarshal([]byte(charactersJson), &characters)
		if err != nil {
			return SearchResult{}, fmt.Errorf("Got invalid JSON from database")
		}

		var artists []string
		err = json.Unmarshal([]byte(artistsJson), &artists)
		if err != nil {
			return SearchResult{}, fmt.Errorf("Got invalid JSON from database")
		}

		var groups []string
		err = json.Unmarshal([]byte(groupsJson), &groups)
		if err != nil {
			return SearchResult{}, fmt.Errorf("Got invalid JSON from database")
		}

		var languages []string
		err = json.Unmarshal([]byte(languagesJson), &languages)
		if err != nil {
			return SearchResult{}, fmt.Errorf("Got invalid JSON from database")
		}

		capePageId := 0
		err = db.db.QueryRow(`SELECT id FROM DoujinPages WHERE doujin_id = ? AND page_number = 1`, id).Scan(&capePageId)
		if err != sql.ErrNoRows && err != nil {
			return SearchResult{}, err
		}

		pages := [][]int{{1, capePageId}}

		doujins = append(doujins, Doujin{
			Id:             id,
			Title:          title,
			Subtitle:       subtitle,
			UploadDate:     uploadDate,
			ExternalRating: externalRating,
			Tags:           tags,
			Characters:     characters,
			Artists:        artists,
			Groups:         groups,
			Languages:      languages,
			Pages:          pages,
		})
	}

	return SearchResult{
		Entries:    doujins,
		TotalPages: totalPages,
	}, nil
}

func (db *Database) GetDoujinMetadata(username string, token string, id int) (Doujin, error) {
	_, err := db.authenticateUser(username, token)
	if err != nil {
		return Doujin{}, err
	}

	var resultId int
	var resultTitle string
	var resultSubtitle string
	var resultUploadDate string
	var resultExternalRating int
	var resultTagsJson string
	var resultCharactersJson string
	var resultArtistsJson string
	var resultGroupsJson string
	var resultLanguagesJson string

	err = db.db.QueryRow(
		`SELECT id, title, subtitle, upload_date, external_rating, tags, characters, artists, groups, languages
		 FROM Doujins
		 WHERE id = ?`,
		id,
	).Scan(
		&resultId, &resultTitle, &resultSubtitle,
		&resultUploadDate, &resultExternalRating, &resultTagsJson,
		&resultCharactersJson, &resultArtistsJson, &resultGroupsJson,
		&resultLanguagesJson,
	)

	if err == sql.ErrNoRows {
		return Doujin{}, DatabaseErrorInvalidId
	}

	if err != nil {
		return Doujin{}, err
	}

	var resultTags []string
	err = json.Unmarshal([]byte(resultTagsJson), &resultTags)
	if err != nil {
		return Doujin{}, fmt.Errorf("Got invalid JSON from database")
	}

	var resultCharacters []string
	err = json.Unmarshal([]byte(resultCharactersJson), &resultCharacters)
	if err != nil {
		return Doujin{}, fmt.Errorf("Got invalid JSON from database")
	}

	var resultArtists []string
	err = json.Unmarshal([]byte(resultArtistsJson), &resultArtists)
	if err != nil {
		return Doujin{}, fmt.Errorf("Got invalid JSON from database")
	}

	var resultGroups []string
	err = json.Unmarshal([]byte(resultGroupsJson), &resultGroups)
	if err != nil {
		return Doujin{}, fmt.Errorf("Got invalid JSON from database")
	}

	var resultLanguages []string
	err = json.Unmarshal([]byte(resultLanguagesJson), &resultLanguages)
	if err != nil {
		return Doujin{}, fmt.Errorf("Got invalid JSON from database")
	}

	rows, err := db.db.Query(`SELECT id, page_number FROM DoujinPages WHERE doujin_id = ?`, id)
	if err != nil {
		return Doujin{}, err
	}
	defer rows.Close()

	pages := [][]int{}

	for rows.Next() {
		var pageId int
		var pageNumber int

		err = rows.Scan(&pageId, &pageNumber)
		if err != nil {
			return Doujin{}, err
		}

		pages = append(pages, []int{pageNumber, pageId})
	}

	return Doujin{
		Id:             resultId,
		Title:          resultTitle,
		Subtitle:       resultSubtitle,
		UploadDate:     resultUploadDate,
		ExternalRating: resultExternalRating,
		Tags:           resultTags,
		Characters:     resultCharacters,
		Artists:        resultArtists,
		Groups:         resultGroups,
		Languages:      resultLanguages,
		Pages:          pages,
	}, nil
}

func (db *Database) GetAllTags(username string, token string) ([]string, error) {
	_, err := db.authenticateUser(username, token)
	if err != nil {
		return nil, err
	}

	rows, err := db.db.Query(`
		SELECT DISTINCT jt.value AS tag
		FROM Doujins, json_each(Doujins.tags) as jt
		ORDER BY tag
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}

	for rows.Next() {
		var tag string
		err = rows.Scan(&tag)
		if err != nil {
			return nil, err
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

func (db *Database) GetPageFilePath(username string, token string, pageId int) (string, error) {
	_, err := db.authenticateUser(username, token)
	if err != nil {
		return "", err
	}

	var pagePath string
	err = db.db.QueryRow("SELECT page_path FROM DoujinPages WHERE id = ?", pageId).Scan(&pagePath)

	if err == sql.ErrNoRows {
		return "", DatabaseErrorInvalidId
	}

	if err != nil {
		return "", err
	}

	return pagePath, nil
}

func (db *Database) CreateTagSet(username string, token string, tags []string, antiTags []string) (int, error) {
	userId, err := db.authenticateUser(username, token)
	if err != nil {
		return 0, err
	}

	tagsJson := jsonEncode(tags)
	antiTagsJson := jsonEncode(antiTags)

	result, err := sqlExec(
		db.db,
		`INSERT INTO TagSets (user_id, tags, anti_tags) VALUES (?, ?, ?)`,
		userId, tagsJson, antiTagsJson,
	)
	if err != nil {
		return 0, err
	}

	tagSetId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(tagSetId), nil
}

func (db *Database) DeleteTagSet(username string, token string, tagSetId int) error {
	userId, err := db.authenticateUser(username, token)
	if err != nil {
		return err
	}

	var tagSetOwnerId int
	err = db.db.QueryRow(`SELECT user_id FROM TagSets WHERE id = ?`, tagSetId).Scan(&tagSetOwnerId)
	if err == sql.ErrNoRows {
		return DatabaseErrorInvalidId
	}

	if err != nil {
		return err
	}

	if tagSetOwnerId != userId {
		return DatabaseErrorUnauthorized
	}

	_, err = sqlExec(db.db, `DELETE FROM TagSets WHERE id = ?`, tagSetId)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) ChangeTagSet(username string, token string, tagSetId int, tags []string, antiTags []string) error {
	userId, err := db.authenticateUser(username, token)
	if err != nil {
		return err
	}

	var tagSetOwnerId int
	err = db.db.QueryRow(`SELECT user_id FROM TagSets WHERE id = ?`, tagSetId).Scan(&tagSetOwnerId)
	if err == sql.ErrNoRows {
		return DatabaseErrorInvalidId
	}

	if err != nil {
		return err
	}

	if tagSetOwnerId != userId {
		return DatabaseErrorUnauthorized
	}

	tagsJson := jsonEncode(tags)
	antiTagsJson := jsonEncode(antiTags)

	_, err = sqlExec(
		db.db,
		`UPDATE TagSets SET tags = ?, anti_tags = ? WHERE id = ?`,
		tagsJson, antiTagsJson, tagSetId,
	)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetTagSets(username string, token string) ([]TagSet, error) {
	userId, err := db.authenticateUser(username, token)
	if err != nil {
		return nil, err
	}

	rows, err := db.db.Query(`SELECT id, tags, anti_tags FROM TagSets WHERE user_id = ?`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagSets := []TagSet{}

	for rows.Next() {
		var tagSetId int
		var tagSetTagsJson string
		var tagSetAntiTagsJson string

		err = rows.Scan(&tagSetId, &tagSetTagsJson, &tagSetAntiTagsJson)
		if err != nil {
			return nil, err
		}

		var tagSetTags []string
		err = json.Unmarshal([]byte(tagSetTagsJson), &tagSetTags)
		if err != nil {
			return nil, fmt.Errorf("Got invalid JSON from database")
		}

		var tagSetAntiTags []string
		err = json.Unmarshal([]byte(tagSetAntiTagsJson), &tagSetAntiTags)
		if err != nil {
			return nil, fmt.Errorf("Got invalid JSON from database")
		}

		tagSets = append(tagSets, TagSet{
			Id:       tagSetId,
			Tags:     tagSetTags,
			AntiTags: tagSetAntiTags,
		})
	}

	return tagSets, nil
}

func (db *Database) IsAuthDataValid(username string, token string) (bool, error) {
	_, err := db.authenticateUser(username, token)
	if err == DatabaseErrorInexistentUser || err == DatabaseErrorInvalidToken {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (db *Database) LogoutUser(username string, token string) error {
	var sessionTokens SessionTokens
	err := db.db.QueryRow(
		`SELECT session_tokens FROM Users WHERE username = ? COLLATE NOCASE`,
		username,
	).Scan(&sessionTokens)

	if err == sql.ErrNoRows {
		return DatabaseErrorInexistentUser
	}

	if err != nil {
		return err
	}

	err = sessionTokens.RemoveToken(token)
	if err == SessionTokensErrorUnknownToken {
		return DatabaseErrorInvalidToken
	}

	if err != nil {
		return err
	}

	_, err = sqlExec(
		db.db,
		"UPDATE Users SET session_tokens = ? WHERE username = ? COLLATE NOCASE",
		sessionTokens, username,
	)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) Close() {
	db.db.Close()
}
