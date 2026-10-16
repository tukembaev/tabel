package domain

import "time"

// Absence codes used in timesheet reports.
var (
	EmployeeRequestCodes = map[string]struct{}{
		"Б": {}, "БЛ": {}, "Д": {}, "Ж": {}, "О": {}, "ОБС": {}, "Т": {},
		"Учебный отпуск": {}, "Учебный отпуск без содержания": {},
		"Командировка": {}, "Отпуск без содержания по уходу за ребенком": {},
		"Отпуск без содержания": {}, "Отпуск без содержания по инициативе администр": {},
		"Больничный": {}, "Декретный отпуск": {}, "Отпуск": {},
	}
	HeadOnlyCodes = map[string]struct{}{
		"П": {}, "Ув": {}, "ПР": {}, "Прогул": {}, "Уволен": {}, "Праздник": {},
	}
	AllCodes = func() map[string]struct{} {
		m := map[string]struct{}{}
		for k := range EmployeeRequestCodes {
			m[k] = struct{}{}
		}
		for k := range HeadOnlyCodes {
			m[k] = struct{}{}
		}
		m["сб"] = struct{}{}
		m["Воскресение"] = struct{}{}
		m["Суббота"] = struct{}{}
		return m
	}()
)

func IsWeekend(t time.Time) (code string, ok bool) {
	switch t.Weekday() {
	case time.Saturday:
		return "сб", true
	case time.Sunday:
		return "Воскресение", true
	default:
		return "", false
	}
}

func DisplayName(last, first, middle string) string {
	name := last
	if first != "" {
		if name != "" {
			name += " "
		}
		name += first
	}
	if middle != "" {
		if name != "" {
			name += " "
		}
		name += middle
	}
	return name
}
