package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	exec()
}

func exec() {
	bdusss := os.Getenv("BDUSS")
	if bdusss == "" {
		log.Println("环境变量必须设置BDUSS")
	}
	bdussArr := strings.Split(bdusss, "\n")
	c := 0
	rs := []SignTable{}
	for _, bduss := range bdussArr {
		c++
		totalCount := 0
		cookieValidCount := 0
		excepCount := 0
		blackCount := 0
		signCount := 0
		bqCount := 0
		supCount := 0
		if !CheckBdussValid(bduss) {
			log.Println("BDUSS失效")
		} else {
			tbs := GetTbs(bduss)
			likedTbs, _ := GetLikedTiebas(bduss, "")
			totalCount = len(likedTbs)
			for _, tb := range likedTbs {
				signR := SignOneTieBa(tb.Name, tb.Id, bduss, tbs)
				if signR.ErrorCode == "1" {
					cookieValidCount++
				} else if signR.ErrorCode == "340006" || signR.ErrorCode == "300004" {
					//贴吧目录出问题，加载数据失败2
					excepCount++
				} else if signR.ErrorCode == "340008" {
					//黑名单
					blackCount++
				} else if signR.ErrorCode == "0" || signR.ErrorCode == "160002" || signR.ErrorCode == "199901" {
					//签到成功、已经签到、账号封禁，签到不涨经验
					signCount++
				} else if signR.ErrorCode == "2280007" || signR.ErrorCode == "340011" || signR.ErrorCode == "1989004" {
					//签到服务忙、签到过快、数据加载失败1
					//三种情况需要重签
					bqCount += bq(tb.Name, tb.Id, bduss, tbs)
				}
				sup := CelebritySupport(bduss, "", tb.Id, tbs)
				if sup == "已助攻" || sup == "助攻成功" {
					supCount++
				}
			}
			wk := WenKuSign(bduss)
			zd := WenKuSign(bduss)
			profile := GetUserProfile(GetUid(bduss))
			name := jsoniter.Get([]byte(profile), "user").Get("name").ToString()
			nameShow := jsoniter.Get([]byte(profile), "user").Get("name_show").ToString()
			if nameShow != "" {
				name = nameShow
			}
			st := []SignTable{
				{name, totalCount, signCount, bqCount, excepCount, blackCount, wk, zd, supCount},
			}
			rs = append(rs, st[0])
		}
	}
	msgs := GenerateSignResult(0, rs)
	for _, m := range msgs {
		fmt.Println(m + "\n")
	}
	//telegram通知
	TelegramNOtifyResult(GenerateSignResult(1, rs))
}

func TelegramNOtifyResult(msgs []string) {
	token := os.Getenv("TELEGRAM_APITOKEN")
	chectId := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chectId == "" {
		log.Println("如需开启telegram通知，请设置环境变量ELEGRAM_APITOKEN和TELEGRAM_CHAT_ID")
	} else {
		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Panic(err)
		}
		bot.Debug = false
		chectIdInt64, _ := strconv.ParseInt(chectId, 10, 64)
		log.Printf("Authorized on account %s", bot.Self.UserName)
		for _, m := range msgs {
			msg := tgbotapi.NewMessage(chectIdInt64, m)
			bot.Send(msg)
		}

	}
}

func GenerateSignResult(t int, rs []SignTable) []string {
	result := []string{}
	for _, r := range rs {
		s := ""
		if t == 0 {
			s += "贴吧ID:" + "***\n"
		} else {
			s += "贴吧ID:" + r.Name + "\n"
		}
		s += "总数:" + strconv.Itoa(r.Total) + "\n"
		s += "已签到:" + strconv.Itoa(r.Signed) + "\n"
		s += "补签:" + strconv.Itoa(r.Bq) + "\n"
		s += "异常:" + strconv.Itoa(r.Excep) + "\n"
		s += "黑名单:" + strconv.Itoa(r.Black) + "\n"
		s += "名人堂助攻 :" + strconv.Itoa(r.Support) + "\n"
		s += "文库:" + r.Wenku + "\n"
		s += "知道:" + r.Zhidao
		result = append(result, s)
	}
	return result
}

func bq(tbName string, fid string, bduss string, tbs string) int {
	time.Sleep(time.Duration(5) * time.Second)
	signR := SignOneTieBa(tbName, fid, bduss, tbs)
	if signR.ErrorCode == "0" || signR.ErrorCode == "160002" || signR.ErrorCode == "199901" {
		//签到成功、已签到、封禁
		return 1
	} else {
		return 0
	}
}

type SignTable struct {
	Name    string `table:"贴吧ID"`
	Total   int    `table:"总关注数"`
	Signed  int    `table:"已签到"`
	Bq      int    `table:"补签"`
	Excep   int    `table:"异常"`
	Black   int    `table:"黑名单"`
	Wenku   string `table:"文库"`
	Zhidao  string `table:"知道"`
	Support int    `table:"名人堂助攻"`
}

type SignResult struct {
	ErrorCode    string `json:"error_code"`
	ErrorMsg     string `json:"error_msg,omitempty" gorm:"-"`
	SignTime     int64  `json:"signTime" gorm:"column:signTime"`
	SignPoint    string `json:"sign_point" gorm:"-"`
	CountSignNum string `json:"count_sign_num" gorm:"-"`
	Timespan     int64  `json:"timespan" gorm:"-"`
}
type LikedTieba struct {
	Id            string `json:"id,omitempty" gorm:"-"`
	Name          string `json:"name,,omitempty" gorm:"-"`
	Favo_type     string `json:"favo_type" gorm:"-"`
	Level_id      string `json:"level_id"`
	Level_name    string `json:"level_name"`
	Cur_score     string `json:"cur_score"`
	Levelup_score string `json:"levelup_score"`
	Avatar        string `json:"avatar"`
	Slogan        string `json:"slogan"`
}
type LikedApiRep struct {
	ForumList  ForumList `json:"forum_list"`
	HasMore    string    `json:"has_more"`
	ServerTime string    `json:"server_time"`
	Time       int64     `json:"time"`
	Ctime      int       `json:"ctime"`
	Logid      int       `json:"logid"`
	ErrorCore  string    `json:"error_core"`
}

type ForumList struct {
	NonGconforum []LikedTieba `json:"non-gconforum"`
	Gconforum    []LikedTieba `json:"gconforum"`
}

//获取uid
func GetUid(bduss string) string {
	body, _ := Fetch("http://tieba.baidu.com/i/sys/user_json", nil, bduss, "")
	return jsoniter.Get([]byte(body), "id").ToString()
}

//获取tbs
func GetTbs(bduss string) string {
	body, err := Fetch("http://tieba.baidu.com/dc/common/tbs", nil, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	isLogin := jsoniter.Get([]byte(body), "is_login").ToInt()
	if isLogin == 1 {
		return jsoniter.Get([]byte(body), "tbs").ToString()
	}
	return ""
}

//公共贴吧请求（带cookie）
func Fetch(url string, postData map[string]interface{}, bduss string, stoken string) (string, error) {
	return FetchWithHeaders(url, postData, bduss, stoken, nil)
}

func FetchWithHeaders(url string, postData map[string]interface{}, bduss string, stoken string, headers map[string]string) (string, error) {
	var request *http.Request
	httpClient := &http.Client{}
	if nil == postData {
		request, _ = http.NewRequest("GET", url, nil)
	} else {
		postParams := url2.Values{}
		for key, value := range postData {
			postParams.Set(key, value.(string))
		}
		postDataStr := postParams.Encode()
		postDataBytes := []byte(postDataStr)
		postBytesReader := bytes.NewReader(postDataBytes)
		request, _ = http.NewRequest("POST", url, postBytesReader)
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	if "" != bduss {
		request.AddCookie(&http.Cookie{Name: "BDUSS", Value: bduss})
	}
	if "" != stoken {
		request.AddCookie(&http.Cookie{Name: "STOKEN", Value: stoken})
	}
	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}
	response, fetchError := httpClient.Do(request)
	if fetchError != nil {
		return "", fetchError
	}
	defer response.Body.Close()
	body, readError := ioutil.ReadAll(response.Body)
	if readError != nil {
		return "", readError
	}
	return string(body), nil
}

//BDUSS有效性检测
func CheckBdussValid(bduss string) bool {
	body, err := Fetch("http://tieba.baidu.com/dc/common/tbs", nil, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	isLogin := jsoniter.Get([]byte(body), "is_login").ToInt()
	if isLogin == 1 {
		return true
	}
	return false
}

//获取用户关注的所有贴吧
func GetLikedTiebas(bduss string, uid string) ([]LikedTieba, error) {
	pn := 0
	if uid == "" {
		uid = "" //获取uid
	}
	likedTiebaList := make([]LikedTieba, 0)
	for {
		pn++
		var postData = map[string]interface{}{
			"_client_version": "6.2.2",
			"is_guest":        "0",
			"page_no":         strconv.Itoa(pn),
		}
		postData["sign"] = DataSign(postData)
		body, err := Fetch("http://c.tieba.baidu.com/c/f/forum/like", postData, bduss, "")
		if err != nil {
			log.Println("err:", err)
		}
		var likedApiRep LikedApiRep
		if err := jsoniter.Unmarshal([]byte(body), &likedApiRep); err != nil {
			log.Println("err: ", err)
		}
		for _, likeTb := range likedApiRep.ForumList.Gconforum {
			likedTiebaList = append(likedTiebaList, likeTb)
		}
		for _, likeTb := range likedApiRep.ForumList.NonGconforum {
			likedTiebaList = append(likedTiebaList, likeTb)
		}
		if likedApiRep.HasMore == "0" {
			break
		}

	}
	return likedTiebaList, nil
}

//签到一个贴吧
func SignOneTieBa(tbName string, fid string, bduss string, tbs string) SignResult {
	start := time.Now().UnixNano() / 1e6
	var postData = map[string]interface{}{
		"_client_id":      "03-00-DA-59-05-00-72-96-06-00-01-00-04-00-4C-43-01-00-34-F4-02-00-BC-25-09-00-4E-36",
		"_client_type":    "4",
		"_client_version": "1.2.1.17",
		"_phone_imei":     "540b43b59d21b7a4824e1fd31b08e9a6",
		"fid":             fid,
		"kw":              tbName,
		"net_type":        "3",
		"tbs":             tbs,
	}
	postData["sign"] = DataSign(postData)
	body, err := Fetch("http://c.tieba.baidu.com/c/c/forum/sign", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	errorCode := jsoniter.Get([]byte(body), "error_code").ToString()
	errorMsg := jsoniter.Get([]byte(body), "error_msg").ToString()
	userInfo := jsoniter.Get([]byte(body), "user_info")
	signResult := SignResult{}
	if errorCode == "0" {
		//签到成功
		if userInfo == nil {
			signResult.SignPoint = "0"
			signResult.CountSignNum = "0"
		} else {
			signResult.SignPoint = userInfo.Get("sign_bonus_point").ToString()
			signResult.CountSignNum = userInfo.Get("cont_sign_num").ToString()
		}

		errorMsg = "签到成功"
	}
	signResult.SignTime = time.Now().UnixNano() / 1e6
	signResult.ErrorCode = errorCode
	signResult.ErrorMsg = errorMsg
	span := (time.Now().UnixNano() / 1e6) - start
	signResult.Timespan = span
	return signResult
}

//文库签到
func WenKuSign(bduss string) string {
	headers := make(map[string]string)
	headers["Host"] = "wenku.baidu.com"
	headers["Referer"] = "https://wenku.baidu.com/task/browse/daily"
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4184.0 Safari/537.36"
	body, err := FetchWithHeaders("https://wenku.baidu.com/task/submit/signin", nil, bduss, "", headers)
	if err != nil {
		log.Println("err: ", err)
	}
	errorNo := jsoniter.Get([]byte(body), "error_no").ToString()
	if body != "" && (errorNo != "0" || errorNo != "1") {
		return "已签到"
	}
	return "未签到"
}

//文库签到
func ZhiDaoSign(bduss string) string {
	stokenBody, err1 := FetchWithHeaders("https://zhidao.baidu.com", nil, bduss, "", map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4184.0 Safari/537.36"})
	if err1 != nil {
		log.Println("err: ", err1)
	}
	stoken := GetBetweenStr(stokenBody, `"stoken":"`, `",`)
	stoken = Substr(stoken, 10, 32)
	time := time.Now().UnixNano() / 1e6
	s := strconv.FormatInt(time, 10)
	var postData = map[string]interface{}{
		"cm":     "100509",
		"stoken": stoken,
		"utdata": "52,52,15,5,9,12,9,52,12,4,15,13,17,12,13,5,13," + s,
	}
	body, err := Fetch("http://zhidao.baidu.com/submit/user", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	errorNo := jsoniter.Get([]byte(body), "errorNo").ToString()
	if body != "" && (errorNo != "0" || errorNo != "2") {
		return "已签到"
	}
	return "未签到"
}

//获取用户基本信息
func GetUserProfile(uid string) string {
	var postData = map[string]interface{}{
		"_client_version": "6.1.2",
		"has_plist":       "2",
		"need_post_count": "1",
		"uid":             uid,
	}
	postData["sign"] = DataSign(postData)
	body, err := Fetch("http://c.tieba.baidu.com/c/u/user/profile", postData, "", "")
	if err != nil {
		log.Println("err: ", err)
	}
	return body
}

//根据贴吧名称获取fid
func GetFid(tbName string) string {
	fid := ""
	body := Get("http://tieba.baidu.com/f/commit/share/fnameShareApi?ie=utf-8&fname=" + tbName)
	jsonBody := jsoniter.Get([]byte(body))
	if jsonBody.Get("no").ToInt() == 0 {
		fid = jsonBody.Get("data").Get("fid").ToString()
	}
	return fid
}

//贴吧未开放此功能
//名人堂助攻： 已助攻{"no":2280006,"error":"","data":[]}
//名人堂助攻： 助攻成功{"no":0,"error":"","data":[...]}
//未关注此吧{"no":3110004,"error":"","data":[]}
func CelebritySupport(bduss string, tbName string, fid string, tbs string) string {
	if fid == "" && tbName == "" {
		log.Println("至少包含贴吧名字、FID中的一个")
	} else if fid == "" && tbName != "" {
		fid = GetFid(tbName)
	}
	if tbs == "" {
		tbs = GetTbs(bduss)
	}
	postData := map[string]interface{}{"forum_id": fid, "tbs": tbs}
	body, err := Fetch("http://tieba.baidu.com/celebrity/submit/getForumSupport", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	npcInfo := jsoniter.Get([]byte(body), "data", 0).Get("npc_info")
	if npcInfo.Size() > 0 {
		npcId := npcInfo.Get("npc_id").ToString()
		postData["npc_id"] = npcId
		suportResult, _ := Fetch("http://tieba.baidu.com/celebrity/submit/support", postData, bduss, "")
		no := jsoniter.Get([]byte(suportResult)).Get("no").ToInt()
		if no == 3110004 {
			return "未关注此吧"
		} else if no == 2280006 {
			return "已助攻"
		} else if no == 0 {
			return "助攻成功"
		}
		return suportResult
	}
	return "该贴吧未开放此功能"
}

//贴吧参数sing MD5签名
func DataSign(postData map[string]interface{}) string {
	var keys []string
	for key, _ := range postData {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	sign_str := ""
	for _, key := range keys {
		sign_str += fmt.Sprintf("%s=%s", key, postData[key])
	}
	sign_str += "tiebaclient!!!"
	MD5 := md5.New()
	MD5.Write([]byte(sign_str))
	MD5Result := MD5.Sum(nil)
	signValue := make([]byte, 32)
	hex.Encode(signValue, MD5Result)
	return strings.ToUpper(string(signValue))
}
func GetBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str
}
func Between(str, starting, ending string) string {
	s := strings.Index(str, starting)
	if s < 0 {
		return ""
	}
	s += len(starting)
	e := strings.Index(str[s:], ending)
	if e < 0 {
		return ""
	}
	return str[s : s+e]
}

func Substr(str string, start, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}

//http get方法
func Get(url string) string {
	res, _ := http.Get(url)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	return string(body)
}
