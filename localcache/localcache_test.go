package localcache

import (
	"container/list"
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

func (s *localCacheSuite) SetupTest() {
	s.localcache = New().(*localCache)
	s.localcache.cap = mockCap
}

func (s *localCacheSuite) TearDownSuite() {}

func (s *localCacheSuite) TearDownTest() {
	s.localcache = nil
}

func TestLocalcacheSuite(t *testing.T) {
	suite.Run(t, new(localCacheSuite))
}

func (s *localCacheSuite) TestGet() {
	now := timeNow().UnixNano()
	tests := []struct {
		desc      string
		setupTest func()
		checkFunc func()
	}{
		{
			desc: "get value if key existed and not expired",
			setupTest: func() {
				s.localcache.store = map[string]*list.Element{
					"testGet": s.localcache.lrulist.PushFront(&cacheItem{
						key: "testGet",
						val: "test",
						exp: now + 1,
					}),
				}
			},
			checkFunc: func() {
				val, ok := s.localcache.Get("testGet")
				s.Require().Equal(val, "test")
				s.Require().True(ok)
			},
		},
		{
			desc: "get value of different data types",
			setupTest: func() {
				s.localcache.store = map[string]*list.Element{
					"int": s.localcache.lrulist.PushFront(&cacheItem{
						key: "int",
						val: 1,
						exp: now + 1,
					}),
					"float": s.localcache.lrulist.PushFront(&cacheItem{
						key: "float",
						val: 1.0,
						exp: now + 1,
					}),
					"bool": s.localcache.lrulist.PushFront(&cacheItem{
						key: "bool",
						val: true,
						exp: now + 1,
					}),
					"array": s.localcache.lrulist.PushFront(&cacheItem{
						key: "array",
						val: []int{1, 2, 3},
						exp: now + 1,
					}),
				}
			},
			checkFunc: func() {
				val, ok := s.localcache.Get("int")
				s.Require().Equal(val, 1)
				s.Require().True(ok)
				val, ok = s.localcache.Get("float")
				s.Require().Equal(val, 1.0)
				s.Require().True(ok)
				val, ok = s.localcache.Get("bool")
				s.Require().Equal(val, true)
				s.Require().True(ok)
				val, ok = s.localcache.Get("array")
				s.Require().Equal(val, []int{1, 2, 3})
				s.Require().True(ok)
			},
		},
		{
			desc: "get nil if key is not existed",
			setupTest: func() {
				s.localcache.store = map[string]*list.Element{
					"keyExist": s.localcache.lrulist.PushFront(&cacheItem{
						key: "keyExist",
						val: "test",
						exp: now + 1,
					}),
				}
			},
			checkFunc: func() {
				val, ok := s.localcache.Get("keyNotExist")
				s.Require().Nil(val)
				s.Require().False(ok)
			},
		},
		{
			desc: "get nil if key existed but expired",
			setupTest: func() {
				s.localcache.store = map[string]*list.Element{
					"keyExpired": s.localcache.lrulist.PushFront(&cacheItem{
						key: "keyExpired",
						val: "test",
						exp: now - 1,
					}),
				}
			},
			checkFunc: func() {
				val, ok := s.localcache.Get("keyExpired")
				s.Require().Nil(val)
				s.Require().False(ok)
			},
		},
		{
			desc: "update lru if key existed and not expired",
			setupTest: func() {
				s.localcache.store = map[string]*list.Element{
					"keyExistNotExpired1": s.localcache.lrulist.PushFront(&cacheItem{
						key: "keyExistNotExpired1",
						val: "test",
						exp: now + 1,
					}),
					"keyExistNotExpired2": s.localcache.lrulist.PushFront(&cacheItem{
						key: "keyExistNotExpired2",
						val: "test",
						exp: now + 1,
					}),
					"keyExistExpired": s.localcache.lrulist.PushFront(&cacheItem{
						key: "keyExistExpired",
						val: "test",
						exp: now - 1,
					}),
				}
			},
			checkFunc: func() {
				s.localcache.Get("keyExistNotExpired1")
				s.Require().Equal("keyExistNotExpired1", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("keyExistNotExpired2")
				s.Require().Equal("keyExistNotExpired2", s.localcache.lrulist.Front().Value.(*cacheItem).key)
				s.localcache.Get("keyExistExpired")
				s.Require().Equal("keyExistNotExpired2", s.localcache.lrulist.Front().Value.(*cacheItem).key)
			},
		},
	}

	for _, t := range tests {
		if t.setupTest != nil {
			t.setupTest()
		}
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
				s.localcache.Set("setValueSucc", "test")
				ele, ok := s.localcache.store["setValueSucc"]
				s.Require().True(ok)
				s.Require().Equal("test", ele.Value.(*cacheItem).val)
				s.Require().Equal(1, s.localcache.size)
			},
		},
		{
			desc: "set value with different data type succ",
			checkFunc: func() {
				s.localcache.Set("setInt", 2)
				ele, ok := s.localcache.store["setInt"]
				s.Require().True(ok)
				s.Require().Equal(2, ele.Value.(*cacheItem).val)
				s.Require().Equal("setInt", s.localcache.lrulist.Front().Value.(*cacheItem).key)

				s.localcache.Set("setFloat", 3.0)
				ele, ok = s.localcache.store["setFloat"]
				s.Require().True(ok)
				s.Require().Equal(3.0, ele.Value.(*cacheItem).val)
				s.Require().Equal("setFloat", s.localcache.lrulist.Front().Value.(*cacheItem).key)

				s.localcache.Set("setArray", []int{1, 2, 3})
				ele, ok = s.localcache.store["setArray"]
				s.Require().True(ok)
				s.Require().Equal([]int{1, 2, 3}, ele.Value.(*cacheItem).val)
				s.Require().Equal("setArray", s.localcache.lrulist.Front().Value.(*cacheItem).key)
			},
		},
		{
			desc: "set value with same key should overwrite old values and reset ttl",
			checkFunc: func() {
				s.localcache.Set("setOverWrite", "old")
				oldCacheSize := s.localcache.size
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(1 * time.Second) }
				s.localcache.Set("setOverWrite", "new")

				ele, ok := s.localcache.store["setOverWrite"]
				exp := ele.Value.(*cacheItem).exp
				s.Require().True(ok)
				s.Require().Equal("new", ele.Value.(*cacheItem).val)
				s.Require().Equal(exp, timeNow().Add(s.localcache.ttl).UnixNano())
				s.Require().Equal(oldCacheSize, s.localcache.size)
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
				s.localcache.Set("setNotExpired1", "notExp1")
				s.localcache.Set("setNotExpired2", "notExp2")
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(-s.localcache.ttl - 1*time.Second) }
				s.localcache.Set("setExpired1", "exp1")
				timeNow = func() time.Time { return time.Unix(1679932800, 0).Add(-s.localcache.ttl - 2*time.Second) }
				s.localcache.Set("setExpired2", "exp2")
				timeNow = func() time.Time { return time.Unix(1679932800, 0) }
				s.localcache.Set("setNotExpired3", "notExp3")

				_, ok := s.localcache.store["setNotExpired1"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setNotExpired2"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setNotExpired3"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setExpired1"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setExpired2"]
				s.Require().False(ok)
				top := s.localcache.exppq[0]
				s.Require().Equal("setExpired1", top.key)
			},
		},
		{
			desc: "evict lru item when set new item if cache is full and all items are not expired",
			checkFunc: func() {
				s.localcache.Set("setFirst", "1")
				s.localcache.Set("setSecond", "2")
				s.localcache.Set("setThird", "3")
				s.localcache.Set("setFourth", "4")
				s.localcache.Set("setFifth", "5")
				s.localcache.Set("setSixth", "6")

				ele, ok := s.localcache.store["setFirst"]
				s.Require().False(ok)
				s.Require().Nil(ele)
				ele, ok = s.localcache.store["setSecond"]
				s.Require().False(ok)
				s.Require().Nil(ele)
				_, ok = s.localcache.store["setThird"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setFourth"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setFifth"]
				s.Require().True(ok)
				_, ok = s.localcache.store["setSixth"]
				s.Require().True(ok)
			},
		},
	}

	for _, t := range tests {
		t.checkFunc()
	}
}
