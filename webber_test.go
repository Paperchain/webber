package webber

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestPost struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func TestCanGet(t *testing.T) {
	uri := "https://jsonplaceholder.typicode.com/posts"

	params := make(map[string]string)
	params["Id"] = "2"

	wbr := &Request{URI: uri}
	res, err := wbr.Get(params)
	require.Nil(t, err)

	err = res.Read(false)
	require.Nil(t, err)

	var tpp []TestPost
	err = json.Unmarshal(res.Data, &tpp)
	require.Nil(t, err)

	require.NotEmpty(t, tpp)

	for _, tp := range tpp {
		assert.True(t, tp.ID > 0)
		assert.True(t, tp.UserID > 0)
		assert.NotEmpty(t, tp.Title)
		assert.NotEmpty(t, tp.Body)
	}
}

func TestCanPost(t *testing.T) {
	uri := "https://jsonplaceholder.typicode.com/posts"

	tp := TestPost{
		Body:   "Hello body",
		Title:  "Hello title",
		UserID: 1,
	}

	wbr := &Request{URI: uri, ContentType: ContentTypeApplicationJSON}
	res, err := wbr.Post(tp)
	require.Nil(t, err)

	err = res.Read(false)
	require.Nil(t, err)

	err = json.Unmarshal(res.Data, &tp)
	require.Nil(t, err)

	assert.True(t, tp.UserID > 0)
}
