package mzalendo

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/memcache"
)

func getFromCache(c context.Context, cacheKey string) bool {
	_, err := memcache.Get(c, cacheKey)
	if err != nil {
		return false
	}
	return true
}
