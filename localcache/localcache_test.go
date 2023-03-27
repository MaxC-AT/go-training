package localcache

import (
	"container/heap"
	"container/list"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	mockTTL = 1
	mockCap = 4
)

var (
	mockNew = func(items []*cacheItem) *localCache {
		store := make(map[string]*list.Element)
		lrulist := list.New()
		exppq := make(priorityQueue, len(items))
		for i, item := range items {
			exppq[i] = item
			ele := lrulist.PushFront(item)
			store[item.key] = ele
		}
		heap.Init(&exppq)
		return &localCache{
			ttl:     mockTTL * time.Second,
			cap:     mockCap,
			size:    len(items),
			store:   store,
			lrulist: lrulist,
			exppq:   exppq,
		}
	}
)

type localCacheSuite struct {
	suite.Suite
	localcache *localCache
}

func (s *localCacheSuite) SetupSuite() {}

func (s *localCacheSuite) SetupTest() {}

func (s *localCacheSuite) TearDownSuite() {}

func (s *localCacheSuite) TearDownTest() {}

func TestLocalcacheSuite(t *testing.T) {
	suite.Run(t, new(localCacheSuite))
}

func (s *localCacheSuite) TestGetNil() {
	tests := []struct {
		desc string
		key  string
		val  interface{}
		exp  int64
	}{
		{desc: "nil if key not existed", key: "test1", val: "test1", exp: time.Now().Add((mockTTL) * time.Second).UnixNano()},
		{desc: "nil if key had expired", key: "test", val: "test", exp: time.Now().UnixNano()},
	}

	for _, t := range tests {
		items := createCacheItems(
			[][]interface{}{{t.key, t.val, t.exp}},
		)
		s.localcache = mockNew(items)

		val, ok := s.localcache.Get("test")
		s.Require().Nil(val)
		s.Require().False(ok)
	}
}

func (s *localCacheSuite) TestGetValue() {
	exp := time.Now().Add((mockTTL) * time.Second).UnixNano()
	tests := []struct {
		desc        string
		itemConfigs [][]interface{}
	}{
		{
			desc:        "get value if key existed and not expired",
			itemConfigs: [][]interface{}{{"test1", "test1", exp}},
		},
		{
			desc:        "get value of different data types",
			itemConfigs: [][]interface{}{{"test1", "test1", exp}, {"test2", 2, exp}, {"test3", 3.0, exp}, {"test4", true, exp}},
		},
	}

	for _, t := range tests {
		items := createCacheItems(t.itemConfigs)
		s.localcache = mockNew(items)

		for _, item := range items {
			val, ok := s.localcache.Get(item.key)
			s.Require().Equal(item.val, val)
			s.Require().True(ok)
		}
	}
}

func (s *localCacheSuite) TestGetUpdateLRU() {
	exp1 := time.Now().Add((mockTTL) * time.Second).UnixNano()
	exp2 := time.Now().UnixNano()
	tests := []struct {
		desc        string
		itemConfigs [][]interface{}
		checkFunc   func([]*cacheItem)
	}{
		{
			desc:        "update lru if key existed and not expired",
			itemConfigs: [][]interface{}{{"test1", "test1", exp1}, {"test2", "test2", exp1}, {"test3", "test3", exp1}, {"test4", "test4", exp1}},
			checkFunc: func(items []*cacheItem) {
				for _, item := range items {
					s.localcache.Get(item.key)
					s.Require().Equal(item.key, s.localcache.lrulist.Front().Value.(*cacheItem).key)
				}
			},
		},
		{
			desc:        "not update lru if key had expired",
			itemConfigs: [][]interface{}{{"test1", "test1", exp1}, {"test2", "test2", exp2}, {"test3", "test3", exp2}, {"test4", "test4", exp2}},
			checkFunc: func(items []*cacheItem) {
				for _, item := range items {
					s.localcache.Get(item.key)
					s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				}
			},
		},
	}

	for _, t := range tests {
		items := createCacheItems(t.itemConfigs)
		s.localcache = mockNew(items)
		t.checkFunc(items)
	}
}

func (s *localCacheSuite) TestSet() {
	tests := []struct {
		desc        string
		itemConfigs [][]interface{}
		checkFunc   func([][]interface{})
	}{
		{
			desc:        "set value succ",
			itemConfigs: [][]interface{}{{"test1", "test1"}},
			checkFunc: func(configs [][]interface{}) {
				s.localcache = mockNew(nil)
				for _, config := range configs {
					key := config[0].(string)
					val := config[1].(interface{})

					s.localcache.Set(key, val)

					ele, ok := s.localcache.store[key]
					s.Require().True(ok)
					s.Require().Equal(val, ele.Value.(*cacheItem).val)
				}
				s.Require().Equal(1, s.localcache.size)
			},
		},
		{
			desc:        "set value with different data type succ",
			itemConfigs: [][]interface{}{{"test1", "test1"}, {"test2", 2}, {"test3", 3.0}, {"test4", []int{1, 2, 3}}},
			checkFunc: func(configs [][]interface{}) {
				s.localcache = mockNew(nil)
				for _, config := range configs {
					key := config[0].(string)
					val := config[1].(interface{})

					s.localcache.Set(key, val)

					ele, ok := s.localcache.store[key]
					s.Require().True(ok)
					s.Require().Equal(val, ele.Value.(*cacheItem).val)
					s.Require().Equal(key, s.localcache.lrulist.Front().Value.(*cacheItem).key)
				}
				s.Require().Equal(len(configs), s.localcache.size)
			},
		},
		{
			desc:        "set value with same key should overwrite old values and reset ttl",
			itemConfigs: [][]interface{}{{"test1", "test1", time.Now().UnixNano()}},
			checkFunc: func(configs [][]interface{}) {
				items := createCacheItems(configs)
				s.localcache = mockNew(items)
				oldexp := items[0].exp

				s.localcache.Set("test1", "test2")

				ele, ok := s.localcache.store["test1"]
				newexp := ele.Value.(*cacheItem).exp
				s.Require().True(ok)
				s.Require().Equal("test2", ele.Value.(*cacheItem).val)
				s.Require().True(newexp > oldexp+mockTTL*1000000)
				s.Require().Equal(1, s.localcache.size)
			},
		},
		{
			desc:        "set values more than capacity should evict",
			itemConfigs: [][]interface{}{},
			checkFunc: func(configs [][]interface{}) {
				s.localcache = mockNew(nil)
				for i := 0; i < mockCap+1; i++ {
					s.localcache.Set(fmt.Sprintf("test%d", i), i)
				}
				s.Require().Equal(mockCap, s.localcache.size)
				s.Require().Equal(mockCap, len(s.localcache.store))
				s.Require().Equal(mockCap, s.localcache.lrulist.Len())
			},
		},
	}

	for _, t := range tests {
		t.checkFunc(t.itemConfigs)
	}
}

func (s *localCacheSuite) TestSetEvict() {
	exp1 := time.Now().Add((mockTTL) * time.Second).UnixNano()
	exp2 := time.Now().UnixNano()
	tests := []struct {
		desc        string
		itemConfigs [][]interface{}
		checkFunc   func([]*cacheItem)
	}{
		{
			desc:        "evict furthest expired items when set new item if cache is full and some items are expired",
			itemConfigs: [][]interface{}{{"test1", "test1", exp1}, {"test2", "test2", exp2}, {"test3", "test3", exp2 - 1}, {"test4", "test4", exp1}},
			checkFunc: func(items []*cacheItem) {
				s.localcache.Set("test5", "test5")

				ele, ok := s.localcache.store["test3"]
				top := heap.Pop(&s.localcache.exppq).(*cacheItem)
				s.Require().False(ok)
				s.Require().Nil(ele)
				s.Require().Equal("test2", top.key)
			},
		},
		{
			desc:        "evict lru item when set new item if cache is full and all items are not expired",
			itemConfigs: [][]interface{}{},
			checkFunc: func(items []*cacheItem) {
				s.localcache.Set("test1", "test1")
				s.localcache.Set("test2", "test2")
				s.localcache.Set("test3", "test3")
				s.localcache.Set("test4", "test4")
				s.localcache.Set("test5", "test5")

				ele, ok := s.localcache.store["test1"]
				s.Require().False(ok)
				s.Require().Nil(ele)
			},
		},
	}

	for _, t := range tests {
		items := createCacheItems(t.itemConfigs)
		s.localcache = mockNew(items)
		t.checkFunc(items)
	}
}

func createCacheItems(configs [][]interface{}) []*cacheItem {
	items := []*cacheItem{}
	for _, c := range configs {
		item := &cacheItem{
			key: c[0].(string),
			val: c[1],
			exp: c[2].(int64),
		}
		items = append(items, item)
	}
	return items
}
