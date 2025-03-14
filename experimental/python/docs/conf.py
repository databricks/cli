# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

import sys
import os

sys.path.append(os.path.abspath(".."))
sys.path.append(os.path.abspath("ext"))


project = "databricks-bundles"
copyright = "2024, Databricks"
author = "Gleb Kanterov"
release = "beta"

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

# Add any Sphinx extension module names here, as strings. They can be
# extensions coming with Sphinx (named 'sphinx.ext.*') or your custom
# ones.
extensions = [
    "autodoc_databricks_bundles",
    "sphinx.ext.autodoc",
    "sphinx.ext.autosummary",
    "sphinx.ext.intersphinx",
]

autodoc_typehints = "signature"

templates_path = ["_templates"]
exclude_patterns = ["_build", "Thumbs.db", ".DS_Store", "__generated__"]

# -- Options for HTML output -------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#options-for-html-output

html_theme = "alabaster"
html_static_path = ["images"]
html_theme_options = {
    "logo": "databricks-logo.svg",
    "github_user": "databricks",
    "github_repo": "databricks-bundles",
    "description": "databricks-bundles: Python support for Databricks Asset Bundles",
    "fixed_sidebar": "true",
    "logo_text_align": "center",
    "github_button": "true",
    "show_related": "true",
    "sidebar_collapse": "true",
}

python_maximum_signature_line_length = 20

add_module_names = False
python_use_unqualified_type_names = True

# we include all classes by hand, because not everything is documented
autosummary_generate = False

autodoc_default_options = {
    "members": True,
    "member-order": "bysource",
    "undoc-members": True,
}

autoclass_content = "class"

toc_object_entries = False

intersphinx_mapping = {
    "python": ("https://docs.python.org/3.10", None),
}
