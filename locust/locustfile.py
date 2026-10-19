from locust import HttpUser, FastHttpUser, task, between
import random
import json


class ProductUser_OLD(HttpUser):
    wait_time = between(0.1, 0.5)

    @task(3)
    def get_product(self):
        product_id = random.choice([1, 2, 3])
        self.client.get(f"/products/{product_id}", name="/products/[id]")

    @task(1)
    def update_product(self):
        product_id = random.choice([1, 2, 3])
        payload = {
            "sku": f"SKU-{random.randint(1000, 9999)}",
            "manufacturer": "TestCorp",
            "category_id": random.randint(1, 50),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100),
        }
        self.client.post(
            f"/products/{product_id}/details",
            json=payload,
            name="/products/[id]/details",
        )


class ProductUser(FastHttpUser):
    wait_time = between(0.1, 0.5)

    @task(3)
    def get_product(self):
        product_id = random.choice([1, 2, 3])
        self.client.get(f"/products/{product_id}", name="/products/[id]")

    @task(1)
    def update_product(self):
        product_id = random.choice([1, 2, 3])
        payload = {
            "sku": f"SKU-{random.randint(1000, 9999)}",
            "manufacturer": "TestCorp",
            "category_id": random.randint(1, 50),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100),
        }
        self.client.post(
            f"/products/{product_id}/details",
            json=payload,
            name="/products/[id]/details",
        )
