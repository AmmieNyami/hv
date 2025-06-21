# API Documentation

## General Info

The server talks to the client with JSON. When an endpoint accepts parameters, they are specified via JSON, and when the server returns something, it returns JSON (with the exception of `/api/v1/page`).

The fields of the requests vary depending on the endpoint you're sending the request to.

The responses of the server always follow the following structure:

```json
{
    "error_code":   0,
    "error_string": "OK",
    "data":         null
}
```

Where:

- `"error_code"` indicates if an error happened during the processing of the request and which kind of error happened. `0` means no error, `-1` means generic errors (errors NOT related to database activities like logging in users, searching, etc), and any positive number represents an unique error related to database activities;
- `"error_string"` contains the error message that accompanies the error code;
- `"data"` contains generic data returned by the server. Varies depending on the endpoint the request was sent to.

## Endpoints

In the following documentation for each endpoint, "Response format" refers to the `"data"` field of the response.

### Unauthenticated Endpoints

| Endpoint           | Method | Description                                      |
|--------------------|--------|--------------------------------------------------|
| `/api/v1/register` | `POST` | Registers a user with a username and a password. |

Request format:

```json
{
    "username": "AmmieNyami",
    "password": "123"
}
```

Where:

- `"username"` is the user's username. Usernames must have at least one character and can only contain letters (both uppercase and lowercase), numbers, and `.`, `_` and `-`. Usernames are case insensitive, so if another user that has the same username with a different casing already exists, the user won't be created;
- `"password"` is the user's password. Can contain any characters, but must have at least one character.

Response format: `null`.

| Endpoint        | Method | Description     |
|-----------------|--------|-----------------|
| `/api/v1/login` | `POST` | Logs a user in. |

Request format:

```json
{
    "username": "AmmieNyami",
    "password": "123"
}
```

Where:

- `"username"` is the user's username. Usernames must have at least one character and can only contain letters (both uppercase and lowercase), numbers, and `.`, `_` and `-`. Usernames are case insensitive, so if another user that has the same username with a different casing already exists, the user won't be created;
- `"password"` is the user's password. Can contain any characters, but must have at least one character.

Response format: `null`.

After a call to this endpoint, the server sets the cookie `username` to the username used at login and `token` to the user's session token (required for authenticated API calls).

| Endpoint             | Method | Description                      |
|----------------------|--------|----------------------------------|
| `/api/v1/needsLogin` | `POST` | Checks if credentials are valid. |

Request format: `null`.

Response format:

```json
{
    "needs_login": true
}
```

Where:

- `"needs_login"` indicates whether the credentials stored in the cookies `username` and `token` are valid.

### Authenticated Endpoints

All of these require the cookies `username` and `token` to be set in order to work.

| Endpoint         | Method | Description        |
|------------------|--------|--------------------|
| `/api/v1/logout` | `POST` | Logs the user out. |

Request format: `null`.

Response format: `null`.

After a call to this endpoint, the token stored in the cookie `token` gets invalidated and won't be usable for future authentications.

| Endpoint         | Method | Description                         |
|------------------|--------|-------------------------------------|
| `/api/v1/search` | `POST` | Returns results for a search query. |

Request format:

```json
{
    "query": "yume",
    "page_size": 21,
    "page_number": 1,
    "tags": ["yuri", "slice of life"],
    "anti_tags": ["yaoi"]
}
```

Where:

- `"query"` is the search search query. The server will only return results that contain this query in the title or subtitle;
- `"page_size"` is the size of a page of search results. Must be a number between 1 and 100 inclusive. The page size indicates the number of search results the server should return, and multiplying the page size by the page number minus one results in the number of search results the server should skip;
- `"page_number"` is the number of the page of search results that the server should return. Must be greater than 0. Multiplying the page number minus one by the page size results in the number of search results that the server should skip;
- `"tags"` is an array of tags. The server will only return search results that contain the specified tags;
- `"anti_tags"` is an array of tags. The server will only return search results that do NOT contain these tags.

Response format:

```json
{
    "entries": [
        {
            "id": 25565,
            "title": "[AmmieNyami] Yume no Kyouka ~ Fantastical Ecstasy",
            "subtitle": "[AmmieNyami] \u5922\u306E\u72C2\u83EF\u3000\u301C Fantastical Ecstasy",
            "upload_date": "1996-08-15T07:00:50-03:00",
            "external_rating": 69420,
            "tags": ["yuri", "romance", "slice of life"],
            "characters": ["Amane Mitsuda", "Touma Hisui"],
            "artists": ["AmmieNyami"],
            "groups": ["Team Scarlet Reverie"],
            "languages": ["english"],
            "pages": 20
        }
    ],
    "total_pages": 40
}
```

Where:

- `"entries"` is an array of search results. Each search result is a JSON object representing a doujin. The object has the following structure:

    ```json
    {
        "id": 25565,
        "title": "[AmmieNyami] Yume no Kyouka ~ Fantastical Ecstasy",
        "subtitle": "[AmmieNyami] \u5922\u306E\u72C2\u83EF\u3000\u301C Fantastical Ecstasy",
        "upload_date": "1996-08-15T07:00:50-03:00",
        "external_rating": 69420,
        "tags": ["yuri", "romance", "slice of life"],
        "characters": ["Amane Mitsuda", "Touma Hisui"],
        "artists": ["AmmieNyami"],
        "groups": ["Team Scarlet Reverie"],
        "languages": ["english"],
        "pages": [[1, 19132]]
    }
    ```

    Where:

    - `"id"` is the doujin's ID;
    - `"title"` is the doujin's title;
    - `"subtitle"` is the doujin's subtitle;
    - `"upload_date"` is either the date the doujin was uploaded to the external website it was downloaded from or the date the doujin was first published or imported. The the date this field represents depends on the date specified when importing the doujin. The date is in RFC 3339 format;
    - `"external_rating"` is the rating the doujin received in the external website it was downloaded from. It usually represents a number of views, likes, favorites, etc;
    - `"tags"` is an array containing the doujin's tags;
    - `"characters"` is an array containing the doujin's main characters;
    - `"artists"` is an array containing the names of the artists that worked on the doujin;
    - `"groups"` is an array containing the names of the groups that worked on the doujin;
    - `"languages"` is an array containing the languages used in the doujin;
    - `"pages"` is an array containing an array containing the number of the first page of the doujin and its ID, respectively.

- `"total_pages"` is the number of available pages for this search result, based on the page size specified in the request.

| Endpoint         | Method | Description                        |
|------------------|--------|------------------------------------|
| `/api/v1/doujin` | `POST` | Returns the metadata for a doujin. |

Request format:

```json
{
    "doujin_id": 25565
}
```

Where:

- `"doujin_id"` is the ID of the doujin the server should return metadata for.

Response format:

```json
{
    "doujin": {
        "id": 25565,
        "title": "[AmmieNyami] Yume no Kyouka ~ Fantastical Ecstasy",
        "subtitle": "[AmmieNyami] \u5922\u306E\u72C2\u83EF\u3000\u301C Fantastical Ecstasy",
        "upload_date": "1996-08-15T07:00:50-03:00",
        "external_rating": 69420,
        "tags": ["yuri", "romance", "slice of life"],
        "characters": ["Amane Mitsuda", "Touma Hisui"],
        "artists": ["AmmieNyami"],
        "groups": ["Team Scarlet Reverie"],
        "languages": ["english"],
        "pages": [[1, 19132], [2, 19133], [3, 19134], [4, 19135], [5, 19136], [6, 19137], [7, 19138], [8, 19139], [9, 19140], [10, 19141], [11, 19142], [12, 19143], [13, 19144], [14, 19145], [15, 19146], [16, 19147], [17, 19148], [18, 19149], [19, 19150], [20, 19141]]
    }
}
```

Where:

- `"doujin"` is a JSON object representing a doujin. The object has the following structure:

    ```json
    {
        "id": 25565,
        "title": "[AmmieNyami] Yume no Kyouka ~ Fantastical Ecstasy",
        "subtitle": "[AmmieNyami] \u5922\u306E\u72C2\u83EF\u3000\u301C Fantastical Ecstasy",
        "upload_date": "1996-08-15T07:00:50-03:00",
        "external_rating": 69420,
        "tags": ["yuri", "romance", "slice of life"],
        "characters": ["Amane Mitsuda", "Touma Hisui"],
        "artists": ["AmmieNyami"],
        "groups": ["Team Scarlet Reverie"],
        "languages": ["english"],
        "pages": [[1, 19132], [2, 19133], [3, 19134], [4, 19135], [5, 19136], [6, 19137], [7, 19138], [8, 19139], [9, 19140], [10, 19141], [11, 19142], [12, 19143], [13, 19144], [14, 19145], [15, 19146], [16, 19147], [17, 19148], [18, 19149], [19, 19150], [20, 19141]]
    }
    ```

    Where:

    - `"id"` is the doujin's ID;
    - `"title"` is the doujin's title;
    - `"subtitle"` is the doujin's subtitle;
    - `"upload_date"` is either the date the doujin was uploaded to the external website it was downloaded from or the date the doujin was first published or imported. The the date this field represents depends on the date specified when importing the doujin. The date is in RFC 3339 format;
    - `"external_rating"` is the rating the doujin received in the external website it was downloaded from. It usually represents a number of views, likes, favorites, etc;
    - `"tags"` is an array containing the doujin's tags;
    - `"characters"` is an array containing the doujin's main characters;
    - `"artists"` is an array containing the names of the artists that worked on the doujin;
    - `"groups"` is an array containing the names of the groups that worked on the doujin;
    - `"languages"` is an array containing the languages used in the doujin;
    - `"pages"` is an array of arrays containing the doujin's pages. Each array contains the number of the page followed by its ID.

| Endpoint       | Method | Description                           |
|----------------|--------|---------------------------------------|
| `/api/v1/page` | `POST` | Returns image data for a doujin page. |

Request format:

```json
{
    "page_id": 19132
}
```

Where:

- `"page_id"` is the ID of the doujin page the server should return.

Response format:

- when the content type of the response is `application/json`: `null`;
- when the content type of the response is anything else: raw image data, with the `Content-Type` header set according to the image format.

| Endpoint       | Method | Description                                     |
|----------------|--------|-------------------------------------------------|
| `/api/v1/tags` | `POST` | Returns all tags used by all available doujins. |

Request format: `null`.

Response format:

```json
{
    "tags": ["yaoi", "yuri", "romance", "slice of life"]
}
```

Where:

- `"tags"` is an array containing all tags used by all available doujins.

| Endpoint               | Method | Description                                                               |
|------------------------|--------|---------------------------------------------------------------------------|
| `/api/v1/createTagSet` | `POST` | Creates a set of tags and anti-tags for the user currently authenticated. |

Request format:

```json
{
    "tags": ["yuri", "slice of life"],
    "anti_tags": ["yaoi"]
}
```

Where:

- `"tags"` is an array containing the tags the server should store in the new tag set;
- `"anti_tags"` is an array containing the anti-tags the server should store in the new tag set.

Response format:

```json
{
    "tag_set_id": 6969
}
```

Where:

- `"tag_set_id"` is the ID of the newly created tag set.

Tag sets can be used for storing sets of tags and anti-tags that a user constantly uses in searches.

| Endpoint               | Method | Description                                                                       |
|------------------------|--------|-----------------------------------------------------------------------------------|
| `/api/v1/deleteTagSet` | `POST` | Deletes a set of tags and anti-tags  created by the user currently authenticated. |

Request format:

```json
{
    "tag_set_id": 7070
}
```

Where:

- `"tag_set_id"` is the ID of the tag set the server should delete.

Response format: `null`.

| Endpoint               | Method | Description                                                                      |
|------------------------|--------|----------------------------------------------------------------------------------|
| `/api/v1/changeTagSet` | `POST` | Updates a set of tags and anti-tags created by the user currently authenticated. |

Request format:

```json
{
    "tag_set_id": 6969,
    "tags": ["yaoi", "slice of life"],
    "anti_tags": ["yuri"]
}
```

Where:

- `"tag_set_id"` is the ID of the tag set the server should update;
- `"tags"` is an array containing the tags the server is going the replace the existing tags in the specified tag set with;
- `"anti_tags"` is an array containing the anti-tags the server is going the replace the existing anti-tags in the specified tag set with.

Response format: `null`.

| Endpoint             | Method | Description                                                                             |
|----------------------|--------|-----------------------------------------------------------------------------------------|
| `/api/v1/getTagSets` | `POST` | Returns all the sets of tags and anti-tags created by the user currently authenticated. |

Request format: `null`.

Response format:

```json
{
    "tag_sets": [
        {
            "id": 6969,
            "tags": ["yaoi", "slice of life"],
            "anti_tags": ["yuri"]
        }
    ]
}
```

Where:

- `"tag_sets"` is an array of tag sets. Each tag set is a JSON object with the following structure:

    ```json
    {
        "id": 6969,
        "tags": ["yaoi", "slice of life"],
        "anti_tags": ["yuri"]
    }
    ```

    Where:

    - `"id"` is the ID of the tag set;
    - `"tags"` is an array containing all the tags stored in the tag set;
    - `"anti_tags"` is an array containing all the anti-tags stored in the tag set.

| Endpoint              | Method | Description                                               |
|-----------------------|--------| ----------------------------------------------------------|
| `/api/v1/getUsername` | `POST` | Returns the username of the user currently authenticated. |

Request format: `null`.

Response format:

```json
{
    "username": "AmmieNyami"
}
```

Where:

- `"username"` is the username of the user currently authenticated.
