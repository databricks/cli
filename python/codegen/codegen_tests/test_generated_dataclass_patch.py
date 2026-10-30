from codegen.generated_dataclass import generate_dataclass
from codegen.generated_dataclass_patch import quote_recursive_references
from codegen.jsonschema import Property, Schema, SchemaType


def _model(schema_name: str, properties: dict[str, str]):
    return generate_dataclass(
        namespace="bananas",
        schema_name=schema_name,
        schema=Schema(
            type=SchemaType.OBJECT,
            properties={name: Property(ref=ref) for name, ref in properties.items()},
        ),
    )


def _type_names(model) -> set[str]:
    def visit(type_name):
        return {type_name.name}.union(*(visit(p) for p in type_name.parameters))

    out = set()
    for field in model.fields:
        out |= visit(field.type_name) | visit(field.param_type_name)
    return out


def test_quote_recursive_references():
    # Tree -> Node -> Branch -> list[Node]: only the reference back to Node,
    # which is still being visited from the root, gets quoted
    models = {
        "bananas.Tree": _model("bananas.Tree", {"node": "#/$defs/bananas.Node"}),
        "bananas.Node": _model("bananas.Node", {"branch": "#/$defs/bananas.Branch"}),
        "bananas.Branch": _model(
            "bananas.Branch",
            {"nodes": "#/$defs/slice/bananas.Node", "tree_name": "#/$defs/string"},
        ),
    }

    quote_recursive_references(models, root="bananas.Tree")

    assert '"Node"' in _type_names(models["bananas.Branch"])
    assert '"NodeParam"' in _type_names(models["bananas.Branch"])
    assert "Node" in _type_names(models["bananas.Tree"])
    assert "Branch" in _type_names(models["bananas.Node"])
