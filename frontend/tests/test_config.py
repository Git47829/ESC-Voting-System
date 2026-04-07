"""
Smoke tests for frontend/src/config.py configuration.

These tests use the stdlib ast module only, so they can run without
installing any third-party dependencies.
"""

import ast
import os

MAIN_PY = os.path.join(os.path.dirname(__file__), "..", "src", "config.py")


def _module_assignments(source: str) -> dict:
    """Return a mapping of name -> ast.Assign nodes for module-level assignments."""
    tree = ast.parse(source)
    assignments = {}
    for node in tree.body:
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name):
                    assignments[target.id] = node
    return assignments


def _get_environ_get_args(assign_node: ast.Assign):
    """
    If the assignment's value is a Call to os.environ.get(key, default),
    return (key, default) as strings.  Returns (None, None) otherwise.
    """
    value = assign_node.value
    if not isinstance(value, ast.Call):
        return None, None
    func = value.func
    # Accept both `os.environ.get(...)` and `environ.get(...)`
    is_environ_get = (
        isinstance(func, ast.Attribute)
        and func.attr == "get"
        and isinstance(func.value, ast.Attribute)
        and func.value.attr == "environ"
    )
    if not is_environ_get:
        return None, None
    args = value.args
    key = args[0].value if args and isinstance(args[0], ast.Constant) else None
    default = args[1].value if len(args) > 1 and isinstance(args[1], ast.Constant) else None
    return key, default


def test_eurostats_url_defined():
    """EUROSTATS_URL must be defined at module level to prevent NameError."""
    with open(MAIN_PY) as f:
        source = f.read()
    assignments = _module_assignments(source)
    assert "EUROSTATS_URL" in assignments, (
        "EUROSTATS_URL is not defined at module level in config.py; "
        "admin_reset_votes would raise NameError at runtime."
    )


def test_eurostats_url_uses_env_var():
    """EUROSTATS_URL should be read from the EUROSTATS_URL environment variable."""
    with open(MAIN_PY) as f:
        source = f.read()
    assignments = _module_assignments(source)
    assert "EUROSTATS_URL" in assignments, "EUROSTATS_URL not assigned at module level"
    key, _ = _get_environ_get_args(assignments["EUROSTATS_URL"])
    assert key == "EUROSTATS_URL", (
        "EUROSTATS_URL should be populated via os.environ.get('EUROSTATS_URL', ...) "
        "to match project conventions."
    )


def test_eurostats_url_has_default():
    """EUROSTATS_URL must have a sensible default that points to the eurostats service."""
    with open(MAIN_PY) as f:
        source = f.read()
    assignments = _module_assignments(source)
    assert "EUROSTATS_URL" in assignments, "EUROSTATS_URL not assigned at module level"
    _, default = _get_environ_get_args(assignments["EUROSTATS_URL"])
    assert default is not None and "eurostats" in default, (
        f"EUROSTATS_URL default value '{default}' should reference the 'eurostats' "
        "service hostname (e.g. 'http://eurostats:8880')."
    )
