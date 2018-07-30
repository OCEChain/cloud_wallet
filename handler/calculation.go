package handler

import "sync"

//算力管理
type Calculation struct {
	l        sync.RWMutex
	allCal   map[string]float64
	calTotal float64 //总算力
}

var ALlCal *Calculation = new(Calculation)

func init() {
	ALlCal.allCal = make(map[string]float64)
}

//设置算力
func (c *Calculation) SetCal(uid string, cal float64) {
	c.l.Lock()
	defer c.l.Unlock()
	//先减去之前的算力,再加上当前的算力
	prev_cal, ok := c.allCal[uid]
	if !ok {
		prev_cal = 0
	}
	c.calTotal = c.calTotal - prev_cal + cal
	c.allCal[uid] = cal
}

func (c *Calculation) GetCal(uid string) (cal float64) {
	c.l.RLock()
	defer c.l.RUnlock()
	cal, ok := c.allCal[uid]
	if !ok {
		return 0
	}
	return
}

//移除某个用户的算力
func (c *Calculation) DelCal(uid string) {
	c.l.Lock()
	defer c.l.Unlock()
	//先减去之前的算力
	prev_cal, ok := c.allCal[uid]
	if !ok {
		prev_cal = 0
	}
	c.calTotal = c.calTotal - prev_cal
	delete(c.allCal, uid)
}

//获取总算力
func (c *Calculation) GetTotal() (total float64) {
	c.l.RLock()
	defer c.l.RUnlock()
	total = c.calTotal
	return
}
