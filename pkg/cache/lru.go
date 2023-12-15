/*
Copyright 2023 KubeAGI.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cache

import (
	"container/list"
	"fmt"
	"sync"
)

/*
lru(Least Recently Used)

If a data item has been recently accessed or used,
then it has a higher probability of being accessed in the future.

The LRU algorithm maintains a cache where recently accessed data items are stored.
When a new data item needs to be inserted, the LRU algorithm inserts it into the cache and,
if the cache is full, eliminates the least recently used data items to make room for new ones.

Example:

The current cache has 5 elements and the maximum number of elements is also 5.
1, 2, 3, 4, 5

After we access element 3, the order should become:
3, 1, 2, 4, 5

Add a new element 6 and the order should become:
6, 3, 1, 2, 4
*/
type lru struct {
	// ensure thread safety
	m sync.Mutex

	// maximum number of elements
	limit int

	// cache elements in a DoublyLinkedList, allows you to quickly get the value corresponding to an element.
	cache map[any]*list.Element

	// a DoublyLinkedList that stores elements.
	// an element at the end of a DoublyLinkedList that needs to be removed after the maximum number of elements has been reached.
	list *list.List
}

// lruItem the elements inside a DoublyLinkedList,
// where each item of the linkedlist stores the key and value.
type lruItem struct {
	key, val any
}

func NewLRU(limit int) (Cache, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit needs to be greater than 0")
	}
	return &lru{limit: limit, cache: make(map[any]*list.Element), list: list.New(), m: sync.Mutex{}}, nil
}

// Set add or update elements in the cache
func (l *lru) Set(key any, val any) error {
	l.m.Lock()
	defer l.m.Unlock()

	// If the key exists in the linkedlist,
	// then it is an update of the element,
	// and it is also necessary to move the element to the head of the linkedlist
	v, ok := l.cache[key]
	if ok {
		v.Value = lruItem{key: key, val: val}
		l.list.MoveToFront(v)
		return nil
	}

	// If the number of elements reaches the upper limit,
	// you need to remove the elements at the end of the linkedlist,
	// clean up the data stored in the cache.
	if l.list.Len() >= l.limit {
		last := l.list.Back()
		item := last.Value.(lruItem)
		delete(l.cache, item.key)
		l.list.Remove(last)
	}

	// insert new elements into the head of the linkedlist.
	latest := l.list.PushFront(lruItem{key: key, val: val})
	// record the data into the cache.
	l.cache[key] = latest
	return nil
}

// Get try to get the element
func (l *lru) Get(key any) (any, bool) {
	l.m.Lock()
	defer l.m.Unlock()

	// if the key does not exist, just return nonexistent.
	v, ok := l.cache[key]
	if !ok {
		return nil, false
	}

	// If the key exists, we can get the data directly from the cache and at the same time,
	// we need to move the accessed data from the current position of the linkedlist to the head of the linkedlist.
	l.list.MoveToFront(v)
	item := v.Value.(lruItem)
	return item.val, true
}

// Delete delete element
func (l *lru) Delete(key any) error {
	l.m.Lock()
	defer l.m.Unlock()

	// if the key to be deleted does not exist, no processing is required
	v, ok := l.cache[key]
	if !ok {
		// deleting a non-existent element without reporting an error
		return nil
	}

	// simply remove the element from the linkedlist and delete the corresponding value from cache.
	l.list.Remove(v)
	delete(l.cache, key)
	return nil
}
