package main

import "errors"

func (c *Cache) add(item Message) {
	if c.currentItem == maxCachedItems {
		c.currentItem = 0
	}
	c.messages[c.currentItem] = item
	c.currentItem += 1
}

func (c *Cache) get(orderUid string) (Message, error) {
	for _, item := range c.messages {
		if item.OrderUid == orderUid {
			return item, nil
		}
	}
	return Message{}, errors.New("item " + orderUid + " not found in cache")
}
