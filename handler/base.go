package handler

import (
	"crypto/md5"
	rsa_rand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/henrylee2cn/faygo"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

/*
封装各种公共的方法,直接在使用的结构体里面嵌套base使用即可
*/

//私钥
var privateKey string = `
-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDZsfv1qscqYdy4vY+P4e3cAtmvppXQcRvrF1cB4drkv0haU24Y
7m5qYtT52Kr539RdbKKdLAM6s20lWy7+5C0DgacdwYWd/7PeCELyEipZJL07Vro7
Ate8Bfjya+wltGK9+XNUIHiumUKULW4KDx21+1NLAUeJ6PeW+DAkmJWF6QIDAQAB
AoGBAJlNxenTQj6OfCl9FMR2jlMJjtMrtQT9InQEE7m3m7bLHeC+MCJOhmNVBjaM
ZpthDORdxIZ6oCuOf6Z2+Dl35lntGFh5J7S34UP2BWzF1IyyQfySCNexGNHKT1G1
XKQtHmtc2gWWthEg+S6ciIyw2IGrrP2Rke81vYHExPrexf0hAkEA9Izb0MiYsMCB
/jemLJB0Lb3Y/B8xjGjQFFBQT7bmwBVjvZWZVpnMnXi9sWGdgUpxsCuAIROXjZ40
IRZ2C9EouwJBAOPjPvV8Sgw4vaseOqlJvSq/C/pIFx6RVznDGlc8bRg7SgTPpjHG
4G+M3mVgpCX1a/EU1mB+fhiJ2LAZ/pTtY6sCQGaW9NwIWu3DRIVGCSMm0mYh/3X9
DAcwLSJoctiODQ1Fq9rreDE5QfpJnaJdJfsIJNtX1F+L3YceeBXtW0Ynz2MCQBI8
9KP274Is5FkWkUFNKnuKUK4WKOuEXEO+LpR+vIhs7k6WQ8nGDd4/mujoJBr5mkrw
DPwqA3N5TMNDQVGv8gMCQQCaKGJgWYgvo3/milFfImbp+m7/Y3vCptarldXrYQWO
AQjxwc71ZGBFDITYvdgJM1MTqc8xQek1FXn1vfpy2c6O
-----END RSA PRIVATE KEY-----
`

// 解密
func RsaDecrypt(ciphertext []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, errors.New("private key error!")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rsa_rand.Reader, priv, ciphertext)
}

func Md5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

type data_info map[string]interface{}

type jsonData struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func jsonReturn(ctx *faygo.Context, code int, data interface{}, count ...interface{}) (err error) {
	var j_data interface{}
	j_data = return_jonData(code, data, count...)
	return ctx.JSON(200, j_data)
}

func return_jonData(code int, data interface{}, count ...interface{}) data_info {
	json := make(map[string]interface{})
	json["code"] = code
	json["data"] = data
	if len(count) == 1 {
		json["count"] = count[0]
	}
	return json
}

func curl_get(url string) (data jsonData, err error) {
	client := &http.Client{Timeout: time.Second * 5}
	res, err := client.Get(url)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &data)
	return
}

func curl_post(u string, param map[string]string, duration time.Duration) (data jsonData, err error) {
	client := &http.Client{
		Timeout: duration,
	}
	p := url.Values{}
	for k, v := range param {
		p[k] = []string{v}
	}
	resp, err := client.PostForm(u, p)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &data)
	return
}

func MakeMd5(b []byte) string {
	h := md5.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

//保留n位小数
func Tofix(f float64, n int) (res float64, err error) {
	format := "%." + strconv.Itoa(n) + "f"
	float_str := fmt.Sprintf(format, f)
	res, err = strconv.ParseFloat(float_str, 64)
	return
}

func CheckTradepwd(pwd string) (b bool) {
	if m, _ := regexp.MatchString("^[0-9]{6}$", pwd); !m {
		return false
	}
	return true
}
