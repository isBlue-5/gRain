// Package authz 提供权限控制功能
package authz

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/grain-framework/grain/pkg/auth"
)

// ExpressionEngine 权限表达式引擎
type ExpressionEngine struct {
	scanner  *scanner.Scanner
	tokens   []Token
	pos      int
	identity auth.Identity
}

// Token 表达式标记
type Token struct {
	Type    TokenType
	Value   string
	Literal interface{}
	Line    int
	Column  int
}

// TokenType 标记类型
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenOperator
	TokenParen
	TokenDot
	TokenComma
	TokenWhitespace
)

// Operator 操作符
type Operator struct {
	Symbol        string
	Precedence    int
	Associativity string // "left" or "right"
}

// 预定义操作符
var operators = map[string]Operator{
	"==":       {Symbol: "==", Precedence: 7, Associativity: "left"},
	"!=":       {Symbol: "!=", Precedence: 7, Associativity: "left"},
	"<":        {Symbol: "<", Precedence: 6, Associativity: "left"},
	"<=":       {Symbol: "<=", Precedence: 6, Associativity: "left"},
	">":        {Symbol: ">", Precedence: 6, Associativity: "left"},
	">=":       {Symbol: ">=", Precedence: 6, Associativity: "left"},
	"&&":       {Symbol: "&&", Precedence: 2, Associativity: "left"},
	"||":       {Symbol: "||", Precedence: 1, Associativity: "left"},
	"!":        {Symbol: "!", Precedence: 3, Associativity: "right"},
	"in":       {Symbol: "in", Precedence: 7, Associativity: "left"},
	"contains": {Symbol: "contains", Precedence: 7, Associativity: "left"},
}

// NewExpressionEngine 创建新的表达式引擎
func NewExpressionEngine(identity auth.Identity) *ExpressionEngine {
	return &ExpressionEngine{
		scanner:  &scanner.Scanner{},
		tokens:   []Token{},
		pos:      0,
		identity: identity,
	}
}

// Evaluate 评估权限表达式
func (e *ExpressionEngine) Evaluate(expression string, identity auth.Identity) (bool, error) {
	if strings.TrimSpace(expression) == "" {
		return true, nil
	}

	// 词法分析
	if err := e.tokenize(expression); err != nil {
		return false, fmt.Errorf("词法分析失败: %w", err)
	}

	// 语法分析并执行
	result := e.parseExpression()

	// 转换结果为布尔值
	if boolResult, ok := result.(bool); ok {
		return boolResult, nil
	}

	// 尝试将其他类型转换为布尔值
	return e.convertToBool(result), nil
}

// tokenize 词法分析
func (e *ExpressionEngine) tokenize(expression string) error {
	e.scanner.Init(strings.NewReader(expression))
	e.tokens = []Token{}
	e.pos = 0

	for {
		tok := e.scanner.Scan()
		if tok == scanner.EOF {
			break
		}

		tokenText := e.scanner.TokenText()
		token := Token{
			Type:   e.getTokenType(tok),
			Value:  tokenText,
			Line:   e.scanner.Position.Line,
			Column: e.scanner.Position.Column,
		}

		// 处理特殊标记
		switch token.Type {
		case TokenString:
			// 移除引号
			token.Literal = strings.Trim(tokenText, `"'`)
		case TokenNumber:
			if num, err := strconv.ParseFloat(tokenText, 64); err == nil {
				token.Literal = num
			}
		case TokenIdentifier:
			token.Literal = tokenText
		}

		e.tokens = append(e.tokens, token)
	}

	// 后处理：合并双字符操作符
	e.mergeOperators()

	return nil
}

// mergeOperators 合并双字符操作符
func (e *ExpressionEngine) mergeOperators() {
	var newTokens []Token
	i := 0
	for i < len(e.tokens) {
		if i+1 < len(e.tokens) {
			// 检查是否为双字符操作符
			combined := e.tokens[i].Value + e.tokens[i+1].Value
			if combined == "==" || combined == "!=" || combined == "<=" || combined == ">=" || combined == "&&" || combined == "||" {
				// 创建合并后的标记
				mergedToken := Token{
					Type:   TokenOperator,
					Value:  combined,
					Line:   e.tokens[i].Line,
					Column: e.tokens[i].Column,
				}
				newTokens = append(newTokens, mergedToken)
				i += 2 // 跳过两个标记
				continue
			}
		}
		// 添加单个标记
		newTokens = append(newTokens, e.tokens[i])
		i++
	}
	e.tokens = newTokens
}

// getTokenType 获取标记类型
func (e *ExpressionEngine) getTokenType(tok rune) TokenType {
	switch tok {
	case scanner.Ident:
		return TokenIdentifier
	case scanner.String:
		return TokenString
	case scanner.Int, scanner.Float:
		return TokenNumber
	case ' ', '\t', '\n', '\r':
		return TokenWhitespace
	default:
		return TokenOperator
	}
}

// parseExpression 解析表达式
func (e *ExpressionEngine) parseExpression() interface{} {
	return e.parseLogicalOr()
}

// parseLogicalOr 解析逻辑或表达式
func (e *ExpressionEngine) parseLogicalOr() interface{} {
	left := e.parseLogicalAnd()

	for e.pos < len(e.tokens) && e.currentToken().Value == "||" {
		e.consumeToken() // 消费 ||
		right := e.parseLogicalAnd()
		left = e.evaluateOperator("||", left, right)
	}

	return left
}

// parseLogicalAnd 解析逻辑与表达式
func (e *ExpressionEngine) parseLogicalAnd() interface{} {
	left := e.parseEquality()

	for e.pos < len(e.tokens) && e.currentToken().Value == "&&" {
		e.consumeToken() // 消费 &&
		right := e.parseEquality()
		left = e.evaluateOperator("&&", left, right)
	}

	return left
}

// parseEquality 解析相等性表达式
func (e *ExpressionEngine) parseEquality() interface{} {
	left := e.parseRelational()

	for e.pos < len(e.tokens) {
		token := e.currentToken()
		if token.Value == "==" || token.Value == "!=" {
			e.consumeToken()
			right := e.parseRelational()
			left = e.evaluateOperator(token.Value, left, right)
		} else {
			break
		}
	}

	return left
}

// parseRelational 解析关系表达式
func (e *ExpressionEngine) parseRelational() interface{} {
	left := e.parsePrimary()

	for e.pos < len(e.tokens) {
		token := e.currentToken()
		if token.Value == "<" || token.Value == "<=" || token.Value == ">" || token.Value == ">=" {
			e.consumeToken()
			right := e.parsePrimary()
			left = e.evaluateOperator(token.Value, left, right)
		} else {
			break
		}
	}

	return left
}

// parsePrimary 解析基本表达式
func (e *ExpressionEngine) parsePrimary() interface{} {
	if e.pos >= len(e.tokens) {
		return nil
	}

	token := e.currentToken()

	switch token.Type {
	case TokenIdentifier:
		e.consumeToken()
		// 检查是否有属性访问（如 user.role）
		if e.pos < len(e.tokens) && e.currentToken().Value == "." {
			e.consumeToken() // 消费 .
			if e.pos < len(e.tokens) && e.currentToken().Type == TokenIdentifier {
				propertyToken := e.currentToken()
				e.consumeToken()
				return e.resolveIdentifier(token.Value + "." + propertyToken.Value)
			}
		}
		return e.resolveIdentifier(token.Value)
	case TokenString:
		e.consumeToken()
		return token.Literal
	case TokenNumber:
		e.consumeToken()
		return token.Literal
	default:
		if token.Value == "(" {
			e.consumeToken() // 消费 (
			result := e.parseExpression()
			if e.pos < len(e.tokens) && e.currentToken().Value == ")" {
				e.consumeToken() // 消费 )
			}
			return result
		}
	}

	return nil
}

// resolveIdentifier 解析标识符
func (e *ExpressionEngine) resolveIdentifier(name string) interface{} {
	// 解析 user.role 这样的属性访问
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) == 2 && parts[0] == "user" {
			return e.getUserProperty(parts[1])
		}
	}

	// 简单标识符
	return e.getUserProperty(name)
}

// getUserProperty 获取用户属性
func (e *ExpressionEngine) getUserProperty(property string) interface{} {
	if e.identity == nil {
		return nil
	}

	// 首先尝试从 Claims 中获取属性
	if claims := e.identity.GetClaims(); claims != nil {
		if value, exists := claims[property]; exists {
			return value
		}
	}

	// 然后检查预定义的属性
	switch property {
	case "role":
		// 获取用户的第一个角色
		roles := e.identity.GetRoles()
		if len(roles) > 0 {
			return roles[0]
		}
		return "anonymous"
	case "department":
		// 从 Claims 中获取部门信息
		if claims := e.identity.GetClaims(); claims != nil {
			if dept, exists := claims["department"]; exists {
				return dept
			}
		}
		// 根据角色推断部门
		roles := e.identity.GetRoles()
		for _, role := range roles {
			if role == "admin" {
				return "IT"
			}
		}
		return "general"
	case "id":
		return e.identity.GetID()
	case "roles":
		return e.identity.GetRoles()
	case "username":
		return e.identity.GetUsername()
	default:
		return nil
	}
}

// evaluateOperator 执行操作符
func (e *ExpressionEngine) evaluateOperator(op string, left, right interface{}) interface{} {
	switch op {
	case "==":
		return e.equal(left, right)
	case "!=":
		return !e.equal(left, right)
	case "<":
		return e.lessThan(left, right)
	case "<=":
		return e.lessThanOrEqual(left, right)
	case ">":
		return e.greaterThan(left, right)
	case ">=":
		return e.greaterThanOrEqual(left, right)
	case "&&":
		return e.convertToBool(left) && e.convertToBool(right)
	case "||":
		return e.convertToBool(left) || e.convertToBool(right)
	case "in":
		return e.contains(right, left)
	case "contains":
		return e.contains(left, right)
	default:
		return false
	}
}

// equal 相等比较
func (e *ExpressionEngine) equal(left, right interface{}) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

// lessThan 小于比较
func (e *ExpressionEngine) lessThan(left, right interface{}) bool {
	leftNum, leftOk := e.toNumber(left)
	rightNum, rightOk := e.toNumber(right)
	if leftOk && rightOk {
		return leftNum < rightNum
	}
	return false
}

// lessThanOrEqual 小于等于比较
func (e *ExpressionEngine) lessThanOrEqual(left, right interface{}) bool {
	leftNum, leftOk := e.toNumber(left)
	rightNum, rightOk := e.toNumber(right)
	if leftOk && rightOk {
		return leftNum <= rightNum
	}
	return false
}

// greaterThan 大于比较
func (e *ExpressionEngine) greaterThan(left, right interface{}) bool {
	leftNum, leftOk := e.toNumber(left)
	rightNum, rightOk := e.toNumber(right)
	if leftOk && rightOk {
		return leftNum > rightNum
	}
	return false
}

// greaterThanOrEqual 大于等于比较
func (e *ExpressionEngine) greaterThanOrEqual(left, right interface{}) bool {
	leftNum, leftOk := e.toNumber(left)
	rightNum, rightOk := e.toNumber(right)
	if leftOk && rightOk {
		return leftNum >= rightNum
	}
	return false
}

// contains 包含检查
func (e *ExpressionEngine) contains(container, item interface{}) bool {
	switch c := container.(type) {
	case []string:
		if itemStr, ok := item.(string); ok {
			for _, s := range c {
				if s == itemStr {
					return true
				}
			}
		}
	case string:
		if itemStr, ok := item.(string); ok {
			return strings.Contains(c, itemStr)
		}
	}
	return false
}

// toNumber 转换为数字
func (e *ExpressionEngine) toNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

// convertToBool 转换为布尔值
func (e *ExpressionEngine) convertToBool(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "0"
	case int, int64, float64:
		return v != 0
	default:
		return true
	}
}

// currentToken 获取当前标记
func (e *ExpressionEngine) currentToken() Token {
	if e.pos >= len(e.tokens) {
		return Token{Type: TokenEOF}
	}
	return e.tokens[e.pos]
}

// consumeToken 消费当前标记
func (e *ExpressionEngine) consumeToken() {
	e.pos++
}

// peekToken 查看下一个标记
func (e *ExpressionEngine) peekToken() Token {
	if e.pos+1 >= len(e.tokens) {
		return Token{Type: TokenEOF}
	}
	return e.tokens[e.pos+1]
}
