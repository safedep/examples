package main

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
)

type sqlVisitor interface {
	VisitSelect(stmt *sqlparser.Select)
	VisitWhere(stmt *sqlparser.Where)
	VisitSelectExpr(expr sqlparser.SelectExpr)
	VisitComparisonExpr(expr *sqlparser.ComparisonExpr)
}

type sqlVisitorImpl struct{}

func (v *sqlVisitorImpl) Visit(stmt sqlparser.Statement) {
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		v.VisitSelect(stmt)
	}
}

func (v *sqlVisitorImpl) VisitSelect(stmt *sqlparser.Select) {
	for _, expr := range stmt.SelectExprs {
		v.VisitSelectExpr(expr)
	}

	for _, expr := range stmt.From {
		v.VisitTableExpr(expr)
	}

	if stmt.Where != nil {
		v.VisitWhere(stmt.Where)
	}
}

func (v *sqlVisitorImpl) VisitTableExpr(expr sqlparser.TableExpr) {
	fmt.Println("Visiting table expression")

	switch expr := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		v.VisitAliasedTableExpr(expr)
	default:
		fmt.Printf("Visiting unknown table expression: %v\n", expr)
	}
}

func (v *sqlVisitorImpl) VisitAliasedTableExpr(expr *sqlparser.AliasedTableExpr) {
	fmt.Printf("Visiting aliased table expression: %s\n", expr.Expr.(sqlparser.TableName).Name)
}

func (v *sqlVisitorImpl) VisitWhere(stmt *sqlparser.Where) {
	fmt.Println("Visiting WHERE clause")
	v.VisitExpr(stmt.Expr)
}

func (v *sqlVisitorImpl) VisitSelectExpr(expr sqlparser.SelectExpr) {
	fmt.Println("Visiting SELECT expression")

	switch expr := expr.(type) {
	case *sqlparser.StarExpr:
		fmt.Println("Visiting star expression")
	case *sqlparser.AliasedExpr:
		v.VisitAliasedExpr(expr)
	default:
		fmt.Printf("Visiting unknown SELECT expression: %v\n", expr)
	}
}

func (v *sqlVisitorImpl) VisitAliasedExpr(expr *sqlparser.AliasedExpr) {
	fmt.Printf("Visiting aliased expression: %v\n", expr.Expr)

	switch expr := expr.Expr.(type) {
	case *sqlparser.ColName:
		fmt.Printf("Visiting aliased expression: %s.%s\n", expr.Qualifier.Name, expr.Name)
	default:
		fmt.Printf("Visiting unknown aliased expression: %v\n", expr)
	}
}

func (v *sqlVisitorImpl) VisitOrExpr(expr *sqlparser.OrExpr) {
	fmt.Println("Visiting OR expression")

	v.VisitExpr(expr.Left)
	v.VisitExpr(expr.Right)
}

func (v *sqlVisitorImpl) VisitAndExpr(expr *sqlparser.AndExpr) {
	fmt.Println("Visiting AND expression")

	v.VisitExpr(expr.Left)
	v.VisitExpr(expr.Right)
}

func (v *sqlVisitorImpl) VisitComparisonExpr(expr *sqlparser.ComparisonExpr) {
	fmt.Println("Visiting comparison expression")

	left := expr.Left
	right := expr.Right
	operator := expr.Operator

	fmt.Printf("Left: %s\n", left.(*sqlparser.ColName).Name)
	fmt.Printf("Right: %s\n", right.(*sqlparser.SQLVal).Val)
	fmt.Printf("Operator: %v\n", operator)
}

func (v *sqlVisitorImpl) VisitExpr(expr sqlparser.Expr) {
	switch expr := expr.(type) {
	case *sqlparser.ComparisonExpr:
		v.VisitComparisonExpr(expr)
	case *sqlparser.AndExpr:
		v.VisitAndExpr(expr)
	case *sqlparser.OrExpr:
		v.VisitOrExpr(expr)
	case *sqlparser.ParenExpr:
		v.VisitExpr(expr.Expr)
	case *sqlparser.SQLVal:
		fmt.Printf("Visiting SQL value: %s\n", expr.Val)
	case *sqlparser.ColName:
		fmt.Printf("Visiting column name: %s\n", expr.Name)
	default:
		fmt.Printf("Visiting unknown expression: %v\n", expr)
	}
}

func main() {
	sqlString := `SELECT packages.name, users.name, users.id FROM users WHERE (name = 'admin') OR ((age > 18) AND (age < 30))`

	stmt, err := sqlparser.Parse(sqlString)
	if err != nil {
		fmt.Println(err)
		return
	}

	visitor := &sqlVisitorImpl{}
	visitor.Visit(stmt)
}
