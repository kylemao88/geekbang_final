//
//

package common

import "fmt"

func MoneyToString(money int64) string {
	yuan := money / 100
	jiao := (money % 100) / 10
	fen := ((money % 100) - (jiao * 10)) % 10
	money_str := fmt.Sprintf("%d.%d%d", yuan, jiao, fen)
	return money_str
}
