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
	dateTimeFormat = "" // 日期时间掩码
)

// NewDateTime
//
//	@Description: 创建日期+时间
//	@param t 时间
//	@return Date
func NewDateTime(t time.Time) DateTime {
	t = t.Local()
	s := fmt.Sprintf("%04d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	v, _ := strconv.ParseUint(s, 10, 64)
	return DateTime(v)
}

// AddDays
//
//	@Description: 增减天数
//	@param day 天数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) AddDays(day int) Date {
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
func (d DateTime) AddMonths(month int) Date {
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
func (d DateTime) AddYears(year int) Date {
	t := d.ToTime()
	t = t.AddDate(year, 0, 0)
	return NewDate(t)
}

// AddHours
//
//	@Description: 增减小时数
//	@param hour 小时数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) AddHours(hour int) Date {
	t := d.ToTime()
	t = t.Add(time.Hour * time.Duration(hour))
	return NewDate(t)
}

// AddMinutes
//
//	@Description: 增减分钟数
//	@param minute 分钟数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) AddMinutes(minute int) Date {
	t := d.ToTime()
	t = t.Add(time.Minute * time.Duration(minute))
	return NewDate(t)
}

// AddSeconds
//
//	@Description: 增减秒数
//	@param second 秒数
//	@return Date
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) AddSeconds(second int) Date {
	t := d.ToTime()
	t = t.Add(time.Second * time.Duration(second))
	return NewDate(t)
}

// ToString
//
//	@Description: 根据全局format格式化输出
//	@return string
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) ToString() string {
	if dateTimeFormat == "" {
		dateTimeFormat = qconfig.Get("", "com.dateTimeMask", "yyyy-MM-dd HH:mm:ss")
	}
	return qconvert.DateTime.ToString(d.ToTime(), dateTimeFormat)
}

// ToTime
//
//	@Description: 转为原生时间对象
//	@return time.Time
//
//goland:noinspection GoMixedReceiverTypes
func (d DateTime) ToTime() time.Time {
	if d == 0 {
		return time.Time{}
	}
	str := fmt.Sprintf("%d", d)
	if len(str) != 14 {
		str = str + strings.Repeat("0", 14-len(str))
	}
	year, _ := strconv.Atoi(str[0:4])
	month, _ := strconv.Atoi(str[4:6])
	day, _ := strconv.Atoi(str[6:8])
	hour, _ := strconv.Atoi(str[8:10])
	minute, _ := strconv.Atoi(str[10:12])
	second, _ := strconv.Atoi(str[12:14])
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
}

// MarshalJSON
//
//	@Description: 复写json转换
//	@return []byte
//	@return error
func (d DateTime) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("\"%s\"", d.ToString())
	return []byte(str), nil
}

// UnmarshalJSON
//
//	@Description: 复写json转换
//	@param data
//	@return error
func (d *DateTime) UnmarshalJSON(data []byte) error {
	v, err := qconvert.DateTime.ToTime(string(data))
	if err == nil {
		s := fmt.Sprintf("%04d%02d%02d%02d%02d%02d", v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second())
		t, _ := strconv.ParseUint(s, 10, 64)
		*d = DateTime(t)
	}
	return err
}
