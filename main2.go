package main

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	all []string
	str []string
)

func main() {
	all = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	str = []string{"「幸運を呼ぶ」とも言われる黄金のおたまじゃくしが香川県高松市で見つかりました。カエルになった姿は果たして。モグモグと餌（えさ）を頬張るのは、香川県の四国水族館で展示されているニホンアマガエルのおたまじゃくし。通常のものに比べ、色が黄色いのが分かります。先月10日、高松市に住む親子が家の近くの田んぼで、この黄金に輝くおたまじゃくしを捕まえました。", "6日(木)は、大陸から黄砂が飛んでくる可能性があります。過去7月に日本で黄砂が観測されたことはなく、もし観測されればかなり珍しいことになります。車や洗濯物への付着、健康への被害に注意が必要です。"}
	rand.Seed(time.Now().Unix())
	alls := len(all)
	//fmt.Println(alls)
	rnd := rand.Intn(len(all))
	//fmt.Println(rnd)
	if rnd >= (alls / 2) {
		fmt.Println("uploop")
		uploop(rnd, alls)
	} else {
		fmt.Println("downloop")
		downloop(rnd, alls)
	}
}

func uploop(rnd, alls int) {
	cnt := alls
	for i := rnd; i < len(all); i++ {
		fmt.Println(all[i])
		cnt = cnt - 1
	}

	for i := 0; i < cnt; i++ {
		fmt.Println(all[i])
	}
}

func downloop(rnd, alls int) {
	i := rnd
	if rnd > 0 {
		for {
			fmt.Println(i)
			i = i - 1
			if i == 0 {
				break
			}
		}
	}

	count := alls - rnd - 1
	for i := 0; i <= count; i++ {
		fmt.Println(all[alls-i-1])
	}
}
