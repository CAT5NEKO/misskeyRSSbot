//KoyebやVercel等、環境変数を直接読み込まないといけないサービス向けの修正版

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/mmcdole/gofeed"
)

type Config struct {
	MisskeyHost string `envconfig:"MISSKEY_HOST" required:"true"`
	AuthToken   string `envconfig:"AUTH_TOKEN" required:"true"`
	RSSURL      string `envconfig:"RSS_URL" required:"true"`
}

type MisskeyNote struct {
	Text string `json:"text"`
}

type Cache struct {
	mu         sync.RWMutex
	latestItem time.Time
}

func (c *Cache) getLatestItem() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestItem
}

func (c *Cache) saveLatestItem(published time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latestItem = published
}

func processRSS(config Config, cache *Cache) error {

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(config.RSSURL)
	if err != nil {
		log.Println("RSSのパースが上手くできませんでした。:", err)
		return err
	}

	latestItem := cache.getLatestItem()

	log.Println("Feed Title:", feed.Title)
	log.Println("Feed Description:", feed.Description)
	log.Println("Feed Link:", feed.Link)

	if len(feed.Items) > 0 && feed.Items[0].PublishedParsed != nil {
		newestItem := *feed.Items[0].PublishedParsed

		if newestItem.After(latestItem) {
			err := postToMisskey(config, feed.Items[0])
			if err != nil {
				log.Println("Misskeyの投稿をしくじりました...:", err)
				return err
			} else {
				log.Println("Misskeyに投稿しました。:", feed.Items[0].Title)

				cache.saveLatestItem(newestItem)
			}
		}
	}

	return nil
}

func getLatestItem(cache *Cache) time.Time {

	return cache.getLatestItem()
}

func saveLatestItem(cache *Cache, published time.Time) {

	cache.saveLatestItem(published)
}

func postToMisskey(config Config, item *gofeed.Item) error {

	note := map[string]interface{}{
		"i":          config.AuthToken,
		"text":       fmt.Sprintf("新着ニュース！: %s\n%s", item.Title, item.Link),
		"visibility": "home",
	}

	payload, err := json.Marshal(note)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/api/notes/create", config.MisskeyHost)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.AuthToken)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MisskeyAPIと以下の理由で接続を確立できません: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	fmt.Println("処理を開始しますっ！")
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal("環境変数の読み込みをしくじりました...:", err)
	}

	cache := &Cache{}

	//RSSを取得する間隔です。今回は結構頻繁に更新される事例を想定して短めに持たせているけど、NHKとかだと５分スパンで十分です。
	//分数で指定する場合はtime.Minuteに書き換えてください。
	interval := 30 * time.Second
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			log.Println("最新のRSS情報を取得しています")
			errProcessRSS := processRSS(config, cache)
			if errProcessRSS != nil {
				log.Println("RSSの取得に失敗しました...:", errProcessRSS)
			}
			log.Println("最新のRSS情報を取得しました")
		}
	}
}
