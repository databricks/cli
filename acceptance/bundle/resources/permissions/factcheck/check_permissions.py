import sys
import os
import pprint
import json
import subprocess


cli = os.environ["CLI"]


class Error(Exception):
    pass


def run_json(cmd):
    #print("+ " + " ".join(shlex.quote(x) for x in cmd), file=sys.stderr)
    result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, encoding="utf-8")
    if result.returncode != 0:
        if result.stderr.strip():
            raise Error(result.stderr.strip())
        raise Error(f'{cmd} failed with code {result.returncode}')
    try:
        return json.loads(result.stdout)
    except Exception as ex:
        raise Error(f'{cmd} returned non-json: {ex}\n{result.stdout}')


def test_permissions(target_user, resource_type, resource_id, levels, expected):
    acls = []
    if resource_type == "jobs" and target_user != os.environ["USERNAME"]:
        # make sure we keep IS_OWNER
        acls.append({"service_principal_name": os.environ["USERNAME"], "permission_level": "IS_OWNER"})

    for level in levels.split(","):
        acls.append({"user_name": target_user, "permission_level": level})

    request = {"access_control_list": acls}

    try:
        set_result = run_json([cli, "permissions", "set", resource_type, resource_id, "--json", json.dumps(request)])
    except Error as ex:
        sys.exit(f'{resource_type} {levels} => SET ERROR {ex}\nREQUEST:\n' + pprint.pformat(request))

    get_result = run_json([cli, "permissions", "get", resource_type, resource_id])

    if set_result != get_result:
        print("set response is different from set response")
        print("set:")
        pprint.pprint(set_result)
        print("get:")
        pprint.pprint(get_result)

    resulting_levels = []

    for item in set_result["access_control_list"]:
        if (item.get("user_name") or item.get("service_principal_name")) != target_user:
            continue
        for perm in item["all_permissions"]:
            if perm.get("inherited"):
                continue
            resulting_levels.append(perm["permission_level"])

    resulting_levels = ", ".join(resulting_levels)
    if expected == resulting_levels:
        print(f"{resource_type} {levels} => {resulting_levels}")
    else:
        print(f"{resource_type} {levels} => {resulting_levels}; EXPECTED: {expected}")
        print("REQUEST")
        pprint.pprint(request)
        print("RESPONSE")
        pprint.pprint(set_result)
        print()



def main():
    test_permissions(*sys.argv[1:])


if __name__ == "__main__":
    main()
