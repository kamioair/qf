package qdefine

import (
	"fmt"
	"github.com/kamioair/qf/utils/qconfig"
	"github.com/kamioair/qf/utils/qconvert"
	"strconv"
	"strings"
	"time"
)

var (
	dateFormat = "" // 日期掩码
)

// NewDate
//
//	@Description: 创建日期
//	@param t 时间
//	@return Date
func NewDate(t time.Time) Date {
	t = t.Local()
	s := fmt.Sprintf("%04d%02d%02d", t.Year(), t.Month(), t.Day())
	v, _ := strconv.ParseUint(s, 10, 32)
	return Date(v)
}

// ForTo
//
//	@Description: 自动循环遍历日期，并按日返回每天日期
//	@param endDate 结束日期
//	@param interval 日期间隔
//	@param callback 回调
//	@return int YYYY年+WW周，如 202451
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) ForTo(endDate Date, interval uint, callback func(curr Date)) {
	start := d.ToTime()
	end := endDate.ToTime()

	// 判断正序或逆序
	if start.Before(end) || start.Equal(end) {
		// 正序遍历
		for current := start; !current.After(end); current = current.AddDate(0, 0, int(interval)) {
			callback(NewDate(current))
		}
	} else {
		// 逆序遍历
		for current := start; !current.Before(end); current = current.AddDate(0, 0, -int(interval)) {
			callback(NewDate(current))
		}
	}
}

// YearWeek
//
//	@Description: 获取当前日期所在本年的周数
//	@return int YYYY年+WW周，如 202451
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) YearWeek() int {
	y, w := d.ToTime().ISOWeek()
	str := fmt.Sprintf("%d%d", y, w)
	week, _ := strconv.Atoi(str)
	return week
}

// Week
//
//	@Description: 获取当前日期是周几
//	@return int YYYY年+WW周，如 202451
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) Week() time.Weekday {
	return d.ToTime().Weekday()
}

// CurrentToWeekday
//
//	@Description: 获取当前日期所在本周指定周几的日期
//	@param weekday 需要返回周几
//	@return date 周几日期
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) CurrentToWeekday(weekday time.Weekday) Date {
	now := d.ToTime()
	// 获取当前是周几 (0=周日, 1=周一, ..., 6=周六)
	currentWeekday := int(now.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // 将周日看作一周的第 7 天
	}
	// 计算本周的目标周几的偏移天数
	// 如果目标周几比当前周几大，偏移为正；如果比当前小，偏移为负
	offset := int(weekday) - currentWeekday
	// 计算目标日期
	targetDate := now.AddDate(0, 0, offset)
	return NewDate(targetDate)
}

// AddDays
//
//	@Description: 增减天数
//	@param day 天数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) AddDays(day int) Date {
	t := d.ToTime()
	t = t.AddDate(0, 0, day)
	return NewDate(t)
}

// AddMonths
//
//	@Description: 增减月数
//	@param month 月数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) AddMonths(month int) Date {
	t := d.ToTime()
	t = t.AddDate(0, month, 0)
	return NewDate(t)
}

// AddYears
//
//	@Description: 增减年数
//	@param year 年数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) AddYears(year int) Date {
	t := d.ToTime()
	t = t.AddDate(year, 0, 0)
	return NewDate(t)
}

// ToTime
//
//	@Description: 转为原生时间对象
//	@return time.Time
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) ToTime() time.Time {
	if d == 0 {
		return time.Time{}
	}
	str := fmt.Sprintf("%d", d)
	if len(str) != 8 {
		str = str + strings.Repeat("0", 8-len(str))
	}
	year, _ := strconv.Atoi(str[0:4])
	month, _ := strconv.Atoi(str[4:6])
	day, _ := strconv.Atoi(str[6:8])
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

// ToString
//
//	@Description: 根据全局format格式化输出
//	@return string
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) ToString() string {
	if dateFormat == "" {
		dateFormat = qconfig.Get("", "com.dateFormat", "yyyy-MM-dd")
	}
	return qconvert.DateTime.ToString(d.ToTime(), "yyyy-MM-dd")
}

// MarshalJSON
//
//	@Description: 复写json转换
//	@return []byte
//	@return error
//
//goland:noinspection GoMixedReceiverTypes
func (d Date) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("\"%s\"", d.ToString())
	return []byte(str), nil
}

// UnmarshalJSON
//
//	@Description: 复写json转换
//	@param data
//	@return error
//
//goland:noinspection GoMixedReceiverTypes
func (d *Date) UnmarshalJSON(data []byte) error {
	v, err := qconvert.DateTime.ToTime(string(data))
	if err == nil {
		s := fmt.Sprintf("%04d%02d%02d", v.Year(), v.Month(), v.Day())
		t, _ := strconv.ParseUint(s, 10, 64)
		*d = Date(t)
	}
	return err
}
