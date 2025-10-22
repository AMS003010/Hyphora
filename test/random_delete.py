import populate_kv as populate
import requests
import random

for i in range(populate.NO_OF_ENTRIES//2):
    url = f"http://{populate.HOSTNAME}/del"
    entry = random.choice([x for x in range(populate.NO_OF_ENTRIES)])
    body = {
        "key": f"{populate.KEY_FORMAT}_{entry}"
    }

    try:
        response = requests.post(url, json=body)
        if not (200 <= response.status_code <= 300):
            print("DELETE failed:", entry)
    except Exception as e:
        print("Error", e)