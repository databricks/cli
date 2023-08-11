# Databricks notebook source

hello = dbutils.widgets.get("hello")
dbutils.notebook.exit("Widget parameter 'hello' contains: " + hello)
