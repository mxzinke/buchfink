package repository

import (
	"reflect"
	"time"

	"gorm.io/gorm"
)

// utcTimes normalisiert auch ausdrücklich gesetzte Zeitpunkte. NowFunc allein
// erfasst nur die von GORM erzeugten CreatedAt- und UpdatedAt-Werte.
type utcTimes struct{}

func (utcTimes) Name() string { return "buchfink:utc" }
func (utcTimes) Initialize(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:before_create").Register("buchfink:utc", normalizeTimes); err != nil {
		return err
	}
	return db.Callback().Update().Before("gorm:before_update").Register("buchfink:utc", normalizeTimes)
}

func normalizeTimes(db *gorm.DB) {
	var normalize func(reflect.Value)
	normalize = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if v.IsNil() {
				return
			}
			v = v.Elem()
		}
		if v.Type() == reflect.TypeOf(time.Time{}) {
			if v.CanSet() {
				v.Set(reflect.ValueOf(v.Interface().(time.Time).UTC()))
			}
			return
		}
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				normalize(v.Index(i))
			}
		case reflect.Struct:
			if db.Statement.Schema == nil {
				return
			}
			for _, field := range db.Statement.Schema.Fields {
				if field.DBName == "" {
					continue
				}
				value, _ := field.ValueOf(db.Statement.Context, v)
				switch at := value.(type) {
				case time.Time:
					db.AddError(field.Set(db.Statement.Context, v, at.UTC()))
				case *time.Time:
					if at != nil {
						utc := at.UTC()
						db.AddError(field.Set(db.Statement.Context, v, &utc))
					}
				}
			}
		case reflect.Map:
			for _, key := range v.MapKeys() {
				value := v.MapIndex(key).Interface()
				switch at := value.(type) {
				case time.Time:
					v.SetMapIndex(key, reflect.ValueOf(at.UTC()))
				case *time.Time:
					if at != nil {
						utc := at.UTC()
						v.SetMapIndex(key, reflect.ValueOf(&utc))
					}
				}
			}
		}
	}
	normalize(reflect.ValueOf(db.Statement.Dest))
}
