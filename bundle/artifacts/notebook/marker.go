package notebook

import (
	"bufio"
	"io"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func hasMarker(l workspace.Language, r io.Reader) (bool, error) {
	scanner := bufio.NewScanner(r)
	ok := scanner.Scan()
	if !ok {
		return false, scanner.Err()
	}

	line := strings.TrimSpace(scanner.Text())
	switch l {
	case workspace.LanguagePython:
		return line == "# Databricks notebook source", nil
	case workspace.LanguageScala:
		return line == "// Databricks notebook source", nil
	case workspace.LanguageSql:
		return line == "-- Databricks notebook source", nil
	default:
		panic("language not handled: " + l)
	}
}
