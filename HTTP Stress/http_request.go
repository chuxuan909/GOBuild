package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"bytes"
	"encoding/json"
//	"strings"
	"sync"
	"time"
	"math/rand"
	"sort"
	"flag"
)

/************************************变量*****************************************************************/
//各个服务器请求url地址
const (
	 wechat_url = "https://xxxx.com/H5LSK/H5GameTest"   //微信认证服务器地址
	 h5_cer_url = "http://xxxx.com/LSK/userLoginLSK"		 //H5服务认证地址
	 h5_start_url = "http://xxxx.com/LSK/gameStart"	     //H5游戏开始
	 h5_end_url = "http://xxxx.com/LSK/gameOver"            //H5游戏结束
	 MAX_CHANNEL_CACHE = 2000										//管道最大长度
)

// 测试用的商品id列表
// 虽然建议商品id列表在json中获取，但是在测试中发现商品的多少并不影响测试的结果，因此直接写成常量
var test_prize_list = []int{
	100000,100001,100002,
	100003,100004,100005,
	}

var rwlock sync.RWMutex //有全局变量，通常都会有锁机制

//装载错误的管道
var err_chant = make(chan int)
//装载测试共玩了多少次的管道
var count_chant = make(chan int,MAX_CHANNEL_CACHE)
//存储中奖信息的管道
var bingo_chant = make(chan int,MAX_CHANNEL_CACHE)
//存储未中奖信息的管道
var unbingo_chant = make(chan int,MAX_CHANNEL_CACHE)
//运行时间管道
var time_chant = make(chan int64,MAX_CHANNEL_CACHE)

/************************************变量END**************************************************************/

/************************************主函数***************************************************************/
func main() {
	//接收外部选项参数
	var count int
	flag.IntVar(&count,"c",100,"并发数")
	flag.Parse()

	for i:=212;i<212+count;i++{ //并发数
		go play_game(i)
	}

	//主线程阻塞等待并发线程结束
	Log("请按任意键结束测试...")
	var x string
	fmt.Scanln(&x)
	//fmt.Printf("此次测试一个产生了%d次错误",len(err_chant))
	//time.Sleep(time.Second*10)

	//统计
	fmt.Printf("你一共完了%d次\t中奖 “%d” 次\t未中奖 “%d” 次\t金额不足 “%d” 次\n",len(count_chant),len(bingo_chant),len(unbingo_chant),len(count_chant)-len(bingo_chant)-len(unbingo_chant))
	
	//计算游戏流程的最长时间
	close(time_chant)
	time_list := ChantToSlice(time_chant) 			//统计耗时的管道内容输出到一个切片中
	sort.Ints(time_list)                  			//排序
	//max_time := time_list[len(time_list)-1:][0]   //取出切片中的最大值
	max_time := time_list[len(time_list)-1]	        //取出切片中的最大值
	fmt.Printf("游戏最长耗时： %d毫秒\t\t",max_time)

	//计算游戏流程的评价时间
	avg_time := slice_avg(time_list)
	fmt.Printf("游戏平均耗时： %d毫秒\n",avg_time)
}

/************************************主函数END************************************************************/

/************************************主要功能函数*********************************************************/
//游戏入口主函数，需要一个随机的字符串来初始化或者确定用户的unionid
func play_game(unionid_num int){
	//异常捕获
	defer func() { //注意：异常捕获函数必须要写在开头，否则没用！！！
		//异常捕获匿名函数（自执行函数）
		if r := recover(); r != nil {
				fmt.Printf("捕获到的错误：%v\n", r)
		}
	}()

	t_start := time.Now().UnixNano()
	Log("【H5游戏流程开始】") 			     
	Log(" 微信服务器认证中... ...") 
	uid,code :=HttpPostForm(wechat_url,unionid_num)             //微信服务器认证功能函数，根据发送的unionid来获取uid和code
	Log(" 登陆H5游戏... ...") 
	tocken :=H5_Cer(uid,code)             		    //H5登陆功能函数，根据微信服务返回的结果去H5服务认证,返回认证用的tocken
	Log(" 登陆成功，开始游戏... ...") 
	bingo,prize_id := H5_start(uid,tocken) 			//游戏开始功能函数
	Log(" 游戏完毕，结束中... ...") 
	res_code := H5_end(uid,prize_id,bingo,tocken)   //游戏结束功能函数
	t_compelete := time.Now().UnixNano()
	Log("【H5游戏流程结束，将输出游戏结果：】") 
	if res_code == 0 {
		fmt.Printf("\n 游戏用户uid为:%d,游戏商品id为:%d 是否中奖(1为中奖0为不中奖)? ===>【%d】<===\n",uid,prize_id,bingo)
	}
	count_chant <- 1  //放入管道计数
	time_chant <- (t_compelete-t_start)
}


//去微信服务器认证
func HttpPostForm(url string,unionid_num int) (uid int,codes string){

	/**********************准备post数据*****************************/
	// post_data := make(map[string]map[string]interface{})
	// post_data["userInfo"] = make(map[string]interface{})
	// post_data["userInfo"]["unionid"]= "okdDX1U3eEshOW18t_OEuhNKsOjE"
	// post_data["userInfo"]["sex"]= 1
	// post_data["userInfo"]["nickname"]= "坤测试"
	// post_data["userInfo"]["headimgurl"]= "无"
	// post_data["userInfo"]["openid"]= "o77lw1jv5O_hKMUL3O2lFcBAfbEk"

	// j_data,err := json.Marshal(post_data)
	// Do_err(err)
	// fmt.Println(string(j_data))
	/**********************************************************/	
	
	post_data := fmt.Sprintf("{\"userInfo\":{\"unionid\":\"OW18tasd1%d\",\"sex\":1,\"nickname\":\"坤测试\",\"headimgurl\":\"无\",\"openid\":\"o77lw1jv5O_hKMUL3O2lFcBAfbEk\"}}",unionid_num)

	body:=http_handler(url,post_data,"application/json") //调用辅助函数

	// buf := bytes.Buffer{}
	// buf.WriteString(post_data)
	// posts := bytes.NewBuffer(buf.Bytes())

	// req, err := http.NewRequest("POST", wechat_url, posts)
	// Do_err(err)
	// req.Header.Set("Content-Type", "application/json")

	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
    //     panic(err)
	// }
	
	// defer resp.Body.Close()

    // body, _ := ioutil.ReadAll(resp.Body)
	// //fmt.Println(string(body))
	//Log(string(body))

	//反序列化接收到的结果
	var v interface{}
	json.Unmarshal(body,&v)

	//接口类型转换1
	res_map,ok := v.(map[string]interface{})
	if ok == false {
		panic(ok)
	}
	//接口类型转换2
	get_res_map,ok := res_map["body"].(map[string]interface{})
	if ok == false {
		panic(ok)
	}

	re_uid := get_res_map["uid"].(float64) //接口转换3
	re_codes := get_res_map["code"]

	uid = int(re_uid)
	codes = re_codes.(string)
	return
}
	

//去H5服务器认证
func H5_Cer(uid int,code string) (tocken string){

	post_data := fmt.Sprintf("uid=%d&code=%s&unionId=okdDX1U3eEshOW18t_OEuhNKsOjE&loginIp=127.0.0.1&loginType=1&sex=1&nickName=坤测试&photo=www.baidu.com&openId=2sadsad",uid,code)
	//posts  := strings.NewReader(post_data)

	body:=http_handler(h5_cer_url,post_data,"application/x-www-form-urlencoded") //调用辅助函数

	// posts := bytes.NewBufferString(post_data)
	// req, err := http.NewRequest("POST", h5_cer_url, posts)
	// Do_err(err)
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
    //     panic(err)
	// }
	
	// defer resp.Body.Close()

    // body, _ := ioutil.ReadAll(resp.Body)
	//Log(string(body))

	//反序列化接收到的结果
	var v interface{}
	json.Unmarshal(body,&v)


	//************如何把body收到的json格式层层深入从而获得自己想要的值************//
	//接口类型转换1:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map,ok := v.(map[string]interface{})
	if ok == false {
		panic(ok)
	}

	//接口类型转换2:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map_second,ok := res_map["body"].(map[string]interface{}) //转换字典某个字段里面的interface{}
	if ok == false {
		panic(ok)
	}

	re_token := res_map_second["token"] //获取到了想要的数据，但是类型仍然为接口类型

	tocken = re_token.(string)   //接口类型转换为字符串
	return
}

//模拟H5游戏开始
func H5_start(uid int,tocken string) (bingo,prize_id int){

	//读写锁
	rwlock.RLock()
	rand_bum:=rand_number(len(test_prize_list))
	prize_id = test_prize_list[rand_bum]
	rwlock.RUnlock()

	post_data := fmt.Sprintf("uid=%d&token=%s&prizeId=%d",uid,tocken,prize_id)
	//posts  := strings.NewReader(post_data)

	body:=http_handler(h5_start_url,post_data,"application/x-www-form-urlencoded") //调用辅助函数

	// posts := bytes.NewBufferString(post_data)
	// req, err := http.NewRequest("POST", h5_start_url, posts)
	// Do_err(err)
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
    //     panic(err)
	// }
	
	// defer resp.Body.Close()

    // body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	//Log(string(body))

	//反序列化接收到的结果
	var v interface{}
	json.Unmarshal(body,&v)


	//************如何把body收到的json格式层层深入从而获得自己想要的值************//
	//接口类型转换1:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map,ok := v.(map[string]interface{})
	if ok == false {
		panic(ok)
	}

	//接口类型转换2:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map_second,ok := res_map["body"].(map[string]interface{}) //转换字典某个字段里面的interface{}
	if ok == false {
		panic(ok)
	}

	re_bingo := res_map_second["bingo"] //获取到了想要的数据，但是类型仍然为接口类型

	bingo = int(re_bingo.(float64))    //接口类型为float64，因此需要再转换为int

	//如果中奖，放入中奖的管道中,否则放入未中奖管道
	if bingo == 1 {
		bingo_chant <- bingo
	}else{
		unbingo_chant <- bingo
	}

	return 
}

//模拟H5游戏结束
func H5_end(uid,prize_id,bingo int,tocken string) (code int) {

	post_data := fmt.Sprintf("uid=%d&token=%s&prizeId=%d&bingo=%d&gameId=1",uid,tocken,prize_id,bingo)
	//posts  := strings.NewReader(post_data)

	body:=http_handler(h5_start_url,post_data,"application/x-www-form-urlencoded") //调用辅助函数

	// posts := bytes.NewBufferString(post_data)
	// req, err := http.NewRequest("POST", h5_start_url, posts)
	// Do_err(err)
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
    //     panic(err)
	// }
	
	// defer resp.Body.Close()

    // body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	//Log(string(body))

	//反序列化接收到的结果
	var v interface{}
	json.Unmarshal(body,&v)


	//************如何把body收到的json格式层层深入从而获得自己想要的值************//
	//接口类型转换1:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map,ok := v.(map[string]interface{})
	if ok == false {
		panic(ok)
	}

	//接口类型转换2:转换接口类型为map[string]interface{},后面的interface{}还能转换
	res_map_second,ok := res_map["errCode"].(float64) //转换字典某个字段里面的interface{}
	if ok == false {
		panic(ok)
	}

	code = int(res_map_second)
	return
}

/************************************主要功能函数END******************************************************/

/************************************其他辅助函数*********************************************************/
//错误处理函数
func Do_err(err error) {
    if err !=nil{
		fmt.Println("you have some error ! information is:",err)
	}
}

//输出优化函数
func Log(inter... interface{}){
	for _,v := range inter {
		fmt.Println(v)
	}
}

//生成范围内的随机数
func rand_number(range_num int) int{
	rand_new := rand.New(rand.NewSource(time.Now().UnixNano())) //生成随机数种子
	return rand_new.Intn(range_num)	                            //返回范围内的一个随机数
}

//Http请求函数，返回body
func http_handler(url,post_data,con_type string) (body []byte){
	client := &http.Client{} 
	posts := bytes.NewBufferString(post_data)
	req, err := http.NewRequest("POST", url, posts)
	Do_err(err)

	req.Header.Set("Content-Type", con_type)

	resp, err := client.Do(req)
	if err != nil {
        panic(err)
	}
	defer resp.Body.Close()
	
	body, _ = ioutil.ReadAll(resp.Body)
	return

}

//管道转切片
func ChantToSlice(chans chan int64)[]int{
	if len(chans)==0 {
		panic("chan 不能为空")
	}

	list :=[]int{}

	for x := range chans {
		list=append(list,int(x/1000/1000))
	}
	return list
}

//求管切片内整数的平均数
func slice_avg(list []int)(avg int){
	if len(list)==0 {
		panic("切片不能为空")
	}
	var sum int
	sum = 0
	for _,value := range list{
		sum+=value
	}
	avg = sum/len(list)
	return
}
/************************************其他辅助函数END******************************************************/