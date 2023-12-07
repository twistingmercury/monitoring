import random
import requests
import time

URLS = ["http://localhost:8080/ping", "http://localhost:8080/pong"]

def invoke_random_url():
  """Invokes a random URL from the list of URLs."""
  url = random.choice(URLS)
  response = requests.get(url)
  print(f"Response from {url}: {response.status_code}")

def main():
  """Invokes the random URL client."""
  while True:
    time.sleep(1)
    invoke_random_url()

if __name__ == "__main__":
  main()