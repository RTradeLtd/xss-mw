package xss

// see https://raw.githubusercontent.com/gin-gonic/contrib/master/secure/secure_test.go

import (
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	//"reflect"
	"bytes"
	//"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"
)

type User struct {
	Id       int     `json:"id" form:"id" binding:"required"`
	Flt      float64 `json:"flt" form:"flt"`
	User     string  `json:"user" form:"user"`
	Email    string  `json:"email" form:"email"`
	Password string  `json:"password" form:"password"`
	CreAt    int64   `json:"cre_at" form:"cre_at"`
	Comment  string  `json:"comment" form:"comment"`
}

// Test as Gin Middleware
//func newServer(options Options) *gin.Engine {
func newServer() *gin.Engine {
	r := gin.Default()

	//r.Use(Secure(options))
	//// the xss middleware
	//xssMdlwr := &xss.XssMw{
	////TableWhitelist: []byte,
	////FieldWhitelist []byte,
	////TableFieldWhitelist []byte,
	//}

	var xssMdlwr XssMw
	r.Use(xssMdlwr.RemoveXss())
	// TODO - filter on Response not Request
	//r.Use(xss.FilterXss())

	r.GET("/user/:id", func(c *gin.Context) {
		c.String(200, fmt.Sprintf("%v", c.Request.Body))
	})

	r.PUT("/user", func(c *gin.Context) {
		//fmt.Println(c.Request.Body)
		var user User
		//fmt.Printf("%#v", user)
		err := c.Bind(&user)
		//fmt.Printf("%#v", user)
		if err != nil {
			//fmt.Println(err)
			c.JSON(404, gin.H{"msg": "Bind Failed."})
			return
		}
		c.JSON(200, user)
	})

	r.POST("/user", func(c *gin.Context) {
		//fmt.Println(c.Request.Body)
		//fmt.Println(c.Header.Get("Content-Length"))
		var user User
		//fmt.Printf("%#v", user)
		err := c.Bind(&user)
		//fmt.Printf("%#v", user)
		if err != nil {
			//fmt.Println(err)
			c.JSON(404, gin.H{"msg": "Bind Failed."})
			return
		}
		c.JSON(201, user)
	})

	return r
}

func TestKeepsValuesStripsHtmlOnPost(t *testing.T) {
	// don't want to see log message while running tests
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "!@$%^ASDF"
	cmnt := `<img src=x onerror=alert(0)>`
	cre_at := "1481017167"
	oParams := `{"id":2, "flt":2.345, "user":"` + user + `", "email": "` + email + `", "password":"` + password + `", "comment":"` + cmnt + `", "cre_at":` + cre_at + `}`
	req, _ := http.NewRequest("POST", "/user", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(oParams)))

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`

	cmnt_clnd := `` // malicious markup content stripped

	expect := fmt.Sprintf(expStr, user, email, password, cmnt_clnd, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())
}

func TestKeepsValuesStripsHtmlOnPut(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "!@$%^ASDF"
	cmnt := `>'>\"><img src=x onerror=alert(0)>`
	cre_at := "1481017167"
	oParams := `{"id":2, "flt":2.345, "user":"` + user + `", "email": "` + email + `", "password":"` + password + `", "comment":"` + cmnt + `", "cre_at":` + cre_at + `}`
	req, _ := http.NewRequest("PUT", "/user", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(oParams)))

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`
	cmnt_clnd := `&gt;&#39;&gt;&#34;&gt;` //i.e. >'>">

	expect := fmt.Sprintf(expStr, user, email, password, cmnt_clnd, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())
}

func TestXssSkippedOnNoContentLength(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "!@$%^ASDF"
	cmnt := `<img src=x onerror=alert(0)>`
	cre_at := "1481017167"
	oParams := `{"id":2, "flt":2.345, "user":"` + user + `", "email": "` + email + `", "password":"` + password + `", "comment":"` + cmnt + `", "cre_at":` + cre_at + `}`
	req, _ := http.NewRequest("POST", "/user", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`

	expect := fmt.Sprintf(expStr, user, email, password, cmnt, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())
}

func TestXssSkippedOnGetRequest(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	cmnt := `<img src=x onerror=alert(0)>`
	oParams := `{"id":2, "comment":"` + cmnt + `"}`

	req, _ := http.NewRequest("GET", "/user/2", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(oParams)))

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, `{{"id":2, "comment":"`+cmnt+`"}}`, resp.Body.String())
}

// TODO - conf feature pass in fields to skip
func TestPasswordIsNotFiltered(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "<>!@$%^ASDF<>" // the 'password' keyword is set to not filter out xss
	cmnt := `<script>alert(0)</script>`
	cre_at := "1481017167"
	oParams := `{"id":2, "flt":2.345, "user":"` + user + `", "email": "` + email + `", "password":"` + password + `", "comment":"` + cmnt + `", "cre_at":` + cre_at + `}`
	req, _ := http.NewRequest("POST", "/user", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(oParams)))

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`

	cmnt_clnd := `` // malicious markup content stripped

	expect := fmt.Sprintf(expStr, user, email, password, cmnt_clnd, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())
}

// TODO
// multipart form posts really need to be filtered!
// careful with content body such as files, images, audio files, etc!
// Content-Disposition: form-data; name="comment"
//>'>\"><img src=x onerror=alert(0)>
func TestXssFiltersJSONAContentTypeOnly(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "!@$%^ASDF"
	cmnt := `>'>\"><img src=x onerror=alert(0)>`
	cre_at := "1481017167"

	Oparams := map[string]string{
		"id":       "2",
		"user":     user,
		"flt":      "2.345",
		"email":    email,
		"password": password,
		"comment":  cmnt,
		"cre_at":   cre_at,
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for key, val := range Oparams {
		_ = writer.WriteField(key, val)
	}
	err := writer.Close()
	assert.Nil(t, err)

	boundary := writer.Boundary()
	close_buf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	req, perr := http.NewRequest("POST", "/user", body)
	assert.Nil(t, perr)
	// Set headers for multipart, and Content Length
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)
	req.ContentLength = int64(body.Len()) + int64(close_buf.Len())

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)
	//fmt.Println(resp.Body.String())
	assert.Equal(t, 201, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`

	cmnt_clnd := `>'>\\\"><img src=x onerror=alert(0)>` // left intact

	expect := fmt.Sprintf(expStr, user, email, password, cmnt_clnd, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())

}

// TODO - prove Headers and Other Request fields left intact
// Prove Headers left untouched
// for example
//      req.Header.Add("Authorization", "Bearer "+authToken)
func TestKeepsHeadersIntact(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	fmt.Println("TODO TestKeepsHeadersIntact")
	t.Skip()

	//// we don't want to see log message while running tests
	//log.SetOutput(ioutil.Discard)
	//defer log.SetOutput(os.Stderr)

	s := newServer()

	user := "TestUser"
	email := "testUser@example.com"
	password := "!@$%^ASDF"
	cmnt := `<img src=x onerror=alert(0)>`
	cre_at := "1481017167"
	oParams := `{"id":2, "flt":2.345, "user":"` + user + `", "email": "` + email + `", "password":"` + password + `", "comment":"` + cmnt + `", "cre_at":` + cre_at + `}`
	req, _ := http.NewRequest("POST", "/user", bytes.NewBufferString(oParams))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(oParams)))
	//req.Header.Add("Authorization", "Bearer "+authToken)

	resp := httptest.NewRecorder()
	s.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)
	expStr := `{
            "id":2,
            "flt":2.345,
            "user":"%v",
            "email":"%v",
            "password":"%v",
            "comment":"%v",
            "cre_at":%v
        }`

	cmnt_clnd := `` // malicious markup content stripped

	expect := fmt.Sprintf(expStr, user, email, password, cmnt_clnd, cre_at)
	assert.JSONEq(t, expect, resp.Body.String())
}

// TODO
// prove the 3 types of filtering