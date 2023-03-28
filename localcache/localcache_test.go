package localcache

import (
	"container/heap"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	mockCap = 4
)

type localCacheSuite struct {
	suite.Suite
	localcache *localCache
}

func (s *localCacheSuite) SetupSuite() {
	timeNow = func() time.Time { return time.Unix(1679932800, 0) }
}

func (s *localCacheSuite) SetupTest() {}

func (s *localCacheSuite) TearDownSuite() {}

func (s *localCacheSuite) TearDownTest() {}

func TestLocalcacheSuite(t *testing.T) {
	suite.Run(t, new(localCacheSuite))
}

func (s *localCacheSuite) TestGet() {
	now := timeNow().UnixNano()
	tests := []struct {
		desc        string
		itemConfigs [][]interface{}
		checkFunc   func()
	}{
		{
			desc:        "get value if key existed and not expired",
			itemConfigs: [][]interface{}{{"test1", "test1", now + 1}},
			checkFunc: func() {
				val, ok := s.localcache.Get("test1")
				s.Require().Equal(val, "test1")
				s.Require().True(ok)
			},
		},
		{
			desc:        "get value of different data types",
			itemConfigs: [][]interface{}{{"test1", "test1", now + 1}, {"test2", 2, now + 1}, {"test3", 3.0, now + 1}, {"test4", true, now + 1}},
			checkFunc: func() {
				val, ok := s.localcache.Get("test1")
				s.Require().Equal(val, "test1")
				s.Require().True(ok)
				val, ok = s.localcache.Get("test2")
				s.Require().Equal(val, 2)
				s.Require().True(ok)
				val, ok = s.localcache.Get("test3")
				s.Require().Equal(val, 3.0)
				s.Require().True(ok)
				val, ok = s.localcache.Get("test4")
				s.Require().Equal(val, true)
				s.Require().True(ok)
			},
		},
		{
			desc:        "get nil if key is not existed",
			itemConfigs: [][]interface{}{{"test1", "test1", now + 1}},
			checkFunc: func() {
				val, ok := s.localcache.Get("test2")
				s.Require().Nil(val)
				s.Require().False(ok)
			},
		},
		{
			desc:        "get nil if key existed but expired",
			itemConfigs: [][]interface{}{{"test1", "test1", timeNow().UnixNano() - 1}},
			checkFunc: func() {
				val, ok := s.localcache.Get("test1")
				s.Require().Nil(val)
				s.Require().False(ok)
			},
		},
		{
			desc:        "update lru if key existed and not expired",
			itemConfigs: [][]interface{}{{"test1", "test1", now + 1}, {"test2", "test2", now + 1}, {"test3", "test3", now + 1}, {"test4", "test4", now + 1}},
			checkFunc: func() {
				s.localcache.Get("test1")
				s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test2")
				s.Require().Equal("test2", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test3")
				s.Require().Equal("test3", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test4")
				s.Require().Equal("test4", s.localcache.lrulist.Front().Value.(*cacheItem).key)
			},
		},
		{
			desc:        "not update lru if key had expired",
			itemConfigs: [][]interface{}{{"test1", "test1", now + 1}, {"test2", "test2", now - 1}, {"test3", "test3", now - 1}, {"test4", "test4", now + 1}},
			checkFunc: func() {
				s.localcache.Get("test1")
				s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test2")
				s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test3")
				s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("test4")
				s.Require().Equal("test4", s.localcache.lrulist.Front().Value.(*cacheItem).key)
			},
		},
	}

	for _, t := range tests {
		s.localcache = New()
		s.localcache.cap = mockCap
		s.createCacheItems(t.itemConfigs)
		t.checkFunc()
	}
}

func (s *localCacheSuite) TestSet() {
	tests := []struct {
		desc      string
		checkFunc func()
	}{
		{
			desc: "set value succ",
			checkFunc: func() {
				s.localcache.Set("test1", "test1")

				ele, ok := s.localcache.store["test1"]
				s.Require().True(ok)
				s.Require().Equal("test1", ele.Value.(*cacheItem).val)
				s.Require().Equal(1, s.localcache.size)
			},
		},
		{
			desc: "set value with different data type succ",
			checkFunc: func() {
				s.localcache.Set("test1", "test1")
				ele, ok := s.localcache.store["test1"]
				s.Require().True(ok)
				s.Require().Equal("test1", ele.Value.(*cacheItem).val)
				s.Require().Equal("test1", s.localcache.lrulist.Front().Value.(*cacheItem).key)

				s.localcache.Set("test2", 2)
				ele, ok = s.localcache.store["test2"]
				s.Require().True(ok)
				s.Require().Equal(2, ele.Value.(*cacheItem).val)
				s.Require().Equal("test2", s.localcache.lrulist.Front().Value.(*cacheItem).key)

				s.localcache.Set("test3", 3.0)
				ele, ok = s.localcache.store["test3"]
				s.Require().True(ok)
				s.Require().Equal(3.0, ele.Value.(*cacheItem).val)
				s.Require().Equal("test3", s.localcache.lrulist.Front().Value.(*cacheItem).key)

				s.localcache.Set("test4", []int{1, 2, 3})
				ele, ok = s.localcache.store["test4"]
				s.Require().True(ok)
				s.Require().Equal([]int{1, 2, 3}, ele.Value.(*cacheItem).val)
				s.Require().Equal("test4", s.localcache.lrulist.Front().Value.(*cacheItem).key)
			},
		},
		{
			desc: "set value with same key should overwrite old values and reset ttl",
			checkFunc: func() {
				s.localcache.Set("test1", "test1")
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(1 * time.Second) }
				s.localcache.Set("test1", "test2")

				ele, ok := s.localcache.store["test1"]
				exp := ele.Value.(*cacheItem).exp
				s.Require().True(ok)
				s.Require().Equal("test2", ele.Value.(*cacheItem).val)
				s.Require().Equal(exp, timeNow().Add(s.localcache.ttl).UnixNano())
				s.Require().Equal(1, s.localcache.size)
			},
		},
		{
			desc: "set values more than capacity should evict",
			checkFunc: func() {
				for i := 0; i < s.localcache.cap+1; i++ {
					s.localcache.Set(fmt.Sprintf("test%d", i), i)
				}
				s.Require().Equal(s.localcache.cap, s.localcache.size)
				s.Require().Equal(s.localcache.cap, len(s.localcache.store))
				s.Require().Equal(s.localcache.cap, s.localcache.lrulist.Len())
			},
		},
		{
			desc: "evict furthest expired items when set new item if cache is full and some items are expired",
			checkFunc: func() {
				s.localcache.Set("test1", "test1")
				s.localcache.Set("test2", "test2")
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(-s.localcache.ttl - 1*time.Second) }
				s.localcache.Set("test3", "test3")
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(-s.localcache.ttl - 2*time.Second) }
				s.localcache.Set("test4", "test4")
				timeNow = func() time.Time { return time.Unix(1679932800, 0) }
				s.localcache.Set("test5", "test5")

				_, ok := s.localcache.store["test3"]
				s.Require().True(ok)
				_, ok = s.localcache.store["test5"]
				s.Require().True(ok)
				_, ok = s.localcache.store["test4"]
				s.Require().False(ok)
				top := heap.Pop(&s.localcache.exppq).(*cacheItem)
				s.Require().Equal("test3", top.key)
			},
		},
		{
			desc: "evict lru item when set new item if cache is full and all items are not expired",
			checkFunc: func() {
				s.localcache.Set("test2", "test2")
				s.localcache.Set("test3", "test3")
				s.localcache.Set("test1", "test1")
				s.localcache.Set("test4", "test4")
				s.localcache.Set("test6", "test6")
				s.localcache.Set("test5", "test5")

				ele, ok := s.localcache.store["test2"]
				s.Require().False(ok)
				s.Require().Nil(ele)
				ele, ok = s.localcache.store["test3"]
				s.Require().False(ok)
				s.Require().Nil(ele)
			},
		},
	}

	for _, t := range tests {
		s.localcache = New()
		s.localcache.cap = mockCap
		t.checkFunc()
	}
}

func (s *localCacheSuite) createCacheItems(configs [][]interface{}) []*cacheItem {
	items := []*cacheItem{}
	for _, c := range configs {
		key := c[0].(string)
		val := c[1]
		exp := c[2].(int64)
		item := s.localcache.setItem(key, val, exp)
		items = append(items, item)
	}
	return items
}
