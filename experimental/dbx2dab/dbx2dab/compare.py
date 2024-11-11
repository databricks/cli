from typing import Dict, List


def recursive_intersection(
    dict1: Dict[str, any], dict2: Dict[str, any]
) -> Dict[str, any]:
    """
    Compute the recursive intersection of two dictionaries.

    Args:
        dict1 (dict): First dictionary.
        dict2 (dict): Second dictionary.

    Returns:
        dict: A new dictionary containing the intersection.
    """
    intersection = {}
    for key in dict1:
        if key in dict2:
            value1 = dict1[key]
            value2 = dict2[key]

            if isinstance(value1, dict) and isinstance(value2, dict):
                nested = recursive_intersection(value1, value2)
                if nested:
                    intersection[key] = nested
            elif isinstance(value1, list) and isinstance(value2, list):
                common = intersect_lists(value1, value2)
                if common:
                    intersection[key] = common
            else:
                if value1 == value2:
                    intersection[key] = value1
    return intersection


def lists_have_key(list1: List[any], list2: List[any], key: str) -> bool:
    """
    Check if two lists contain job clusters.

    Args:
        list1 (list): First list.
        list2 (list): Second list.

    Returns:
        bool: True if both lists contain job clusters, False otherwise.
    """
    keys1 = [item.get(key) if isinstance(item, dict) else None for item in list1]
    keys2 = [item.get(key) if isinstance(item, dict) else None for item in list2]
    return all(keys1) and all(keys2)


def intersect_lists_with_key(list1: List[any], list2: List[any], key: str):
    result = []
    for item1 in list1:
        for item2 in list2:
            if item1.get(key) == item2.get(key):
                nested = recursive_intersection(item1, item2)
                if nested:
                    result.append(nested)
    return result


def subtract_lists_with_key(list1: List[any], list2: List[any], key: str):
    result = []
    for item1 in list1:
        found = False
        for item2 in list2:
            if item1.get(key) == item2.get(key):
                found = True
                nested = recursive_subtract_dict(item1, item2)
                if nested:
                    out = {key: item1[key]}
                    out.update(nested)
                    result.append(out)

        if not found:
            result.append(item1)
    return result


def are_job_cluster_lists(list1: List[any], list2: List[any]) -> bool:
    return lists_have_key(list1, list2, "job_cluster_key")


def are_task_lists(list1: List[any], list2: List[any]) -> bool:
    return lists_have_key(list1, list2, "task_key")


def intersect_lists(list1: List[any], list2: List[any]):
    """
    Compute the intersection of two lists, handling dictionaries within lists.

    Args:
        list1 (list): First list.
        list2 (list): Second list.

    Returns:
        list: A list containing the intersecting elements.
    """
    result = []

    if lists_have_key(list1, list2, "task_key"):
        return intersect_lists_with_key(list1, list2, "task_key")

    if lists_have_key(list1, list2, "job_cluster_key"):
        return intersect_lists_with_key(list1, list2, "job_cluster_key")

    # Generic intersection
    for item1, item2 in zip(list1, list2):
        if item1 is None or item2 is None:
            break

        if isinstance(item1, dict) and isinstance(item2, dict):
            if recursive_compare(item1, item2):
                result.append(item1)
        else:
            if item1 == item2:
                result.append(item1)

    return result


def recursive_compare(d1: Dict[str, any], d2: Dict[str, any]):
    """
    Recursively compare two dictionaries for equality.

    Args:
        d1 (dict): First dictionary.
        d2 (dict): Second dictionary.

    Returns:
        bool: True if dictionaries are equal, False otherwise.
    """
    if d1.keys() != d2.keys():
        return False
    for key in d1:
        v1 = d1[key]
        v2 = d2[key]
        if isinstance(v1, dict) and isinstance(v2, dict):
            if not recursive_compare(v1, v2):
                return False
        elif isinstance(v1, list) and isinstance(v2, list):
            if not intersect_lists(v1, v2) == v1:
                return False
        else:
            if v1 != v2:
                return False
    return True


def recursive_subtract(o1: any, o2: any) -> any:
    """
    Compute the recursive subtraction of two objects.

    Args:
        o1 (any): First object.
        o2 (any): Second object.

    Returns:
        any: The subtracted object.
    """
    if isinstance(o1, dict) and isinstance(o2, dict):
        return recursive_subtract_dict(o1, o2)
    elif isinstance(o1, list) and isinstance(o2, list):
        return recursive_subtract_list(o1, o2)
    else:
        raise ValueError("Unsupported types for subtraction")


def recursive_subtract_dict(
    dict1: Dict[str, any], dict2: Dict[str, any]
) -> Dict[str, any]:
    """
    Compute the recursive subtraction of two dictionaries.

    Args:
        dict1 (dict): First dictionary.
        dict2 (dict): Second dictionary.

    Returns:
        dict: A new dictionary containing the subtraction.
    """
    subtraction = {}
    for key in dict1:
        if key not in dict2:
            subtraction[key] = dict1[key]
        else:
            value1 = dict1[key]
            value2 = dict2[key]

            if isinstance(value1, dict) and isinstance(value2, dict):
                nested = recursive_subtract(value1, value2)
                if nested:
                    subtraction[key] = nested
            elif isinstance(value1, list) and isinstance(value2, list):
                common = recursive_subtract_list(value1, value2)
                if common:
                    subtraction[key] = common
            else:
                if value1 != value2:
                    subtraction[key] = value1
    return subtraction


def recursive_subtract_list(list1: List[any], list2: List[any]):
    """
    Compute the subtraction of two lists, handling dictionaries within lists.

    Args:
        list1 (list): First list.
        list2 (list): Second list.

    Returns:
        list: A list containing the subtracted elements.
    """
    result = []

    if lists_have_key(list1, list2, "task_key"):
        return subtract_lists_with_key(list1, list2, "task_key")

    if lists_have_key(list1, list2, "job_cluster_key"):
        return subtract_lists_with_key(list1, list2, "job_cluster_key")

    # If both lists contain job clusters, compute the subtraction
    # where the job cluster keys are equal.
    if are_job_cluster_lists(list1, list2):
        for item1 in list1:
            found = False
            for item2 in list2:
                if item1.get("job_cluster_key") == item2.get("job_cluster_key"):
                    found = True
                    nested = recursive_subtract_dict(item1, item2)
                    if nested:
                        result.append(nested)

            if not found:
                result.append(item1)
        return result

    for item1, item2 in zip(list1, list2):
        if item1 is None or item2 is None:
            break

        if isinstance(item1, dict) and isinstance(item2, dict):
            if not recursive_compare(item1, item2):
                result.append(item1)
        else:
            if item1 != item2:
                result.append(item1)

    return result


class Walker:
    _insert_callback = None
    _update_callback = None
    _delete_callback = None

    def __init__(
        self, insert_callback=None, update_callback=None, delete_callback=None
    ):
        self._insert_callback = insert_callback
        self._update_callback = update_callback
        self._delete_callback = delete_callback

    def insert_callback(self, path, key, value):
        if self._insert_callback:
            self._insert_callback(path, key, value)
            return
        raise ValueError(f"Insert: {path}: {key}={value}")

    def update_callback(self, path, old_value, new_value):
        if self._update_callback:
            self._update_callback(path, old_value, new_value)
            return
        raise ValueError(f"Update: {path}: {old_value} -> {new_value}")

    def delete_callback(self, path, key, value):
        if self._delete_callback:
            self._delete_callback(path, key, value)
            return
        raise ValueError(f"Delete: {path}: {key}={value}")

    def walk(self, o1, o2, path=None):
        if path is None:
            path = []

        if isinstance(o1, dict) and isinstance(o2, dict):
            return self._walk_dict(o1, o2, path)
        elif isinstance(o1, list) and isinstance(o2, list):
            return self._walk_list(o1, o2, path)
        else:
            return self._walk_scalar(o1, o2, path)

    def _walk_dict(self, o1, o2, path):
        for key in o1:
            if key not in o2:
                self.delete_callback(path, key, o1[key])
            else:
                self.walk(o1[key], o2[key], path + [key])

        for key in o2:
            if key not in o1:
                self.insert_callback(path, key, o2[key])

    def _walk_list(self, o1, o2, path):
        for i, item in enumerate(o1):
            if i >= len(o2):
                self.delete_callback(path, i, item)
            else:
                self.walk(item, o2[i], path + [i])

        for i in range(len(o1), len(o2)):
            self.insert_callback(path, i, o2[i])

    def _walk_scalar(self, o1, o2, path):
        if o1 != o2:
            self.update_callback(path, o1, o2)


def walk(o1, o2, insert_callback=None, update_callback=None, delete_callback=None):
    walker = Walker(insert_callback, update_callback, delete_callback)
    walker.walk(o1, o2)


def recursive_merge(o1: any, o2: any) -> any:
    """
    Compute the recursive merge of two objects.

    Args:
        o1 (any): First object.
        o2 (any): Second object.

    Returns:
        any: The merged object.
    """
    if isinstance(o1, dict) and isinstance(o2, dict):
        return recursive_merge_dict(o1, o2)
    elif isinstance(o1, list) and isinstance(o2, list):
        return recursive_merge_list(o1, o2)
    else:
        raise ValueError("Unsupported types for merge")


def recursive_merge_dict(
    dict1: Dict[str, any], dict2: Dict[str, any]
) -> Dict[str, any]:
    """
    Compute the recursive merge of two dictionaries.

    Args:
        dict1 (dict): First dictionary.
        dict2 (dict): Second dictionary.

    Returns:
        dict: A new dictionary containing the merge.
    """
    merged = dict(dict1)
    for key in dict2:
        if key in merged:
            value1 = dict1[key]
            value2 = dict2[key]

            if isinstance(value1, dict) and isinstance(value2, dict):
                merged[key] = recursive_merge(value1, value2)
            elif isinstance(value1, list) and isinstance(value2, list):
                merged[key] = recursive_merge_list(value1, value2)
            else:
                merged[key] = value2
        else:
            merged[key] = dict2[key]
    return merged


def recursive_merge_list(list1: List[any], list2: List[any]):
    """
    Compute the merge of two lists, handling dictionaries within lists.

    Args:
        list1 (list): First list.
        list2 (list): Second list.

    Returns:
        list: A list containing the merged elements.
    """
    merged = list(list1)
    for i, item in enumerate(list2):
        if i < len(merged):
            merged[i] = recursive_merge(merged[i], item)
        else:
            merged.append(item)
    return merged
