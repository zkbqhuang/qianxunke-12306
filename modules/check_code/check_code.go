package check_code

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"qianxunke-12306/common/http_util"
	"qianxunke-12306/config/api"
	"qianxunke-12306/modules/conversation"
	"strconv"
	"time"
)

type VCode struct {
	Image             string // 验证码
	Timestamp         string // 请求验证码时的时间戳
	CallbackParameter string // 请求验证码时验证时间戳的回调参数
}

type CodeResponse struct {
	Image          string
	Result_message string
	Result_code    string
}

type CheckResponse struct {
	Code    int64
	Data    []string
	Massage string
}

//获取验证码
func getVCode(conversation *conversation.Conversation) (vCode VCode, err error) {
	defer func() {
		if re := recover(); re != nil {
			if err == nil {
				err = errors.New(fmt.Sprintf("%v", re))
			}
		}
	}()
	vCode = VCode{}
	//生成随机数
	cbpara := "jQuery1910"
	for i := 0; i < 16; i++ {
		cbpara += strconv.Itoa(rand.Intn(9))
	}
	vCode.CallbackParameter = cbpara + "_" + strconv.FormatInt(time.Now().UnixNano()/10000000, 10)
	vCode.Timestamp = strconv.FormatInt(time.Now().UnixNano()/10000000, 10)
	req, _ := http.NewRequest(http.MethodGet, api.GET_CHECK_CODE+"?login_site=E&module=login&rand=sjrand&"+vCode.Timestamp+"=&callback="+vCode.CallbackParameter+"&_="+vCode.Timestamp, nil)
	http_util.SetReqHeader(req)
	resp, err := conversation.Client.Do(req)
	if err != nil {
		return
	}
	conversation.C = resp.Cookies()
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error: %s", err)
		return
	}
	str := string(body)
	codeResponse := &CodeResponse{}
	err = http_util.TimeStrapStringJsonToBean(str, codeResponse)
	if err != nil {
		log.Printf("error: %s", err)
		return
	}
	if codeResponse.Result_code != "0" {
		err = errors.New(codeResponse.Result_message)
	}
	vCode.Image = codeResponse.Image
	return
}

//验证码识别
func checkCodeIdentify(vCode VCode) (result string, err error) {
	defer func() {
		if re := recover(); re != nil {
			if err != nil {
				err = errors.New(fmt.Sprintf("%v", re))
			}
		}
	}()
	data := url.Values{}
	data.Add("imageFile", vCode.Image)
	resp, err := http.PostForm(api.POST_CHECK_CODE_FROM_MY_SERVER, data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	checkResponse := CheckResponse{}
	err = json.Unmarshal(body, &checkResponse)
	if err != nil {
		log.Printf("error: %s", err)
		return
	}
	return simulatedClick(checkResponse.Data)
}

//模拟用户点击验证码
func simulatedClick(codeList []string) (result string, err error) {
	if len(codeList) == 0 {
		err = errors.New("codeList is nil")
		return
	}
	offsetsX := "0"
	offsetsY := "0"
	for _, ofset := range codeList {
		switch ofset {
		case "1":
			offsetsY = "40"
			offsetsX = "40"
			break
		case "2":
			offsetsY = "40"
			offsetsX = "110"
			break
		case "3":
			offsetsY = "40"
			offsetsX = "184"
			break
		case "4":
			offsetsY = "40"
			offsetsX = "256"
			break
		case "5":
			offsetsY = "110"
			offsetsX = "40"
			break
		case "6":
			offsetsY = "110"
			offsetsX = "110"
			break
		case "7":
			offsetsY = "110"
			offsetsX = "184"
			break
		case "8":
			offsetsY = "110"
			offsetsX = "256"
			break
		default:
			offsetsY = "-1"
			offsetsX = "-1"
			break
		}
		if len(result) == 0 {
			result += offsetsX + "," + offsetsY
		} else {
			result += "," + offsetsX + "," + offsetsY
		}
	}
	log.Println("result:(" + result + ")")
	return

}

//向12306验证-验证码
func checkCodeTo12306(conversation *conversation.Conversation, code VCode, strIdentify string) (err error) {
	data := url.Values{}
	data.Set("answer", strIdentify)
	data.Set("callback", code.CallbackParameter)
	data.Set("rand", "sjrand")
	data.Set("login_site", "E")
	data.Set("_", code.Timestamp)
	req, _ := http.NewRequest(http.MethodGet, "https://kyfw.12306.cn/passport/captcha/captcha-check?callback="+code.CallbackParameter+"&answer="+strIdentify+"&rand=sjrand&login_site=E&_="+strconv.FormatInt(time.Now().UnixNano()/10000000, 10), nil)
	http_util.AddReqCookie(conversation.C, req)
	http_util.SetReqHeader(req)
	resp, err := conversation.Client.Do(req)
	if err != nil {
		log.Printf("error: %s\n", err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		return
	}
	log.Println(string(body) + "\n")
//	conversation.C = resp.Cookies()
	return
}

func CheckCode(conversation *conversation.Conversation) (code string, err error) {
	vCode, err := getVCode(conversation)
	if err != nil {
		fmt.Printf("获取验证码失败正在重试... %s\n", err.Error())
		return
	}
	result, err := checkCodeIdentify(vCode)
	if err != nil {
		fmt.Printf("验证-验证码失败，开始重试获取验证码...  %s\n", err.Error())
		return
	}
	err = checkCodeTo12306(conversation, vCode, result)
	if err != nil {
		fmt.Printf("将验证码发到12306验证，出错... %s\n", err.Error())
		return
	}
	return result, nil
}
