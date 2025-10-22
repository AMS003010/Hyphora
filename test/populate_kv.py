import requests

HOSTNAME = "192.168.0.4:8081"
NO_OF_ENTRIES = 6
KEY_FORMAT = "qui"

quotes = [
    "It does not do to dwell on dreams and forget to live. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone",
    "There are all kinds of courage. It takes a great deal of bravery to stand up to our enemies, but just as much to stand up to our friends. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone",
    "After all, to the well-organized mind, death is but the next great adventure. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone",
    "It takes a great deal of bravery to stand up to our enemies, but just as much to stand up to our friends. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone",
    "The truth. It is a beautiful and terrible thing, and should therefore be treated with great caution. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone",
    "Fear of a name increases fear of the thing itself. – Hermione Granger, Harry Potter and the Philosopher’s Stone",
    "There are some things you can’t share without ending up liking each other, and knocking out a twelve-foot mountain troll is one of them – J.K. Rowling, Harry Potter and the Philosopher’s Stone",
    "It is our choices, Harry, that show what we truly are, far more than our abilities. – Albus Dumbledore, Harry Potter and the Chamber of Secrets",
    "When in doubt, go to the library. – Ron Weasley, Harry Potter and the Chamber of Secrets",
    "Happiness can be found even in the darkest of times, if one only remembers to turn on the light. – Albus Dumbledore, Harry Potter and the Prisoner of Azkaban",
    "I solemnly swear that I am up to no good. – Harry Potter, Harry Potter and the Prisoner of Azkaban",
    "Mischief Managed. – Harry Potter, Harry Potter and the Prisoner of Azkaban",
    "I am what I am, an’ I’m not ashamed. – Rubeus Hagrid, Harry Potter and the Goblet of Fire",
    "If you want to know what a man’s like, take a good look at how he treats his inferiors, not his equals. – Sirius Black, Harry Potter and the Goblet of Fire",
    "We are only as strong as we are united, as weak as we are divided. – Albus Dumbledore, Harry Potter and the Goblet of Fire",
    "We’ve all got both light and dark inside us. What matters is the part we choose to act on. That’s who we really are. – Sirius Black, Harry Potter and the Order of the Phoenix",
    "Working hard is important. But there is something that matters even more, believing in yourself – Harry Potter, Harry Potter and the Order of the Phoenix",
    "Youth cannot know how age thinks and feels. But old men are guilty if they forget what it was to be young. – Albus Dumbledore, Harry Potter and the Order of the Phoenix",
    "Indifference and neglect often do much more damage than outright dislike. – Albus Dumbledore, Harry Potter and the Order of the Phoenix",
    "You’re just as sane as I am. – Luna Lovegood, Harry Potter and the Order of the Phoenix",
    "Just because you have the emotional range of a teaspoon doesn’t mean we all have. – Hermione Granger, Harry Potter and the Order of the Phoenix",
    "Every human life is worth the same, and worth saving. – Kingsley Shacklebolt, Harry Potter and the Deathly Hallows",
    "Do not pity the dead, Harry. Pity the living, and above all, those who live without love. – Albus Dumbledore, Harry Potter and the Deathly Hallows",
    "Things we lose have a way of coming back to us in the end, if not always in the way we expect. – Luna Lovegood, Harry Potter and the Order of the Phoenix",
    "The ones that love us never really leave us. – Sirius Black, Harry Potter and the Prisoner of Azkaban",
    "It is the unknown we fear when we look upon death and darkness, nothing more – Albus Dumbledore, Harry Potter and the Half-Blood Prince",
    "We are only as strong as we are united, as weak as we are divided. – Albus Dumbledore, Harry Potter and the Goblet of Fire",
    "It is a curious thing, Harry, but perhaps those who are best suited to power are those who have never sought it. – Albus Dumbledore, Harry Potter and the Deathly Hallows",
    "It was, he thought, the difference between being dragged into the arena to face a battle to the death and walking into the arena with your head held high. – J.K. Rowling, Harry Potter and the Half-Blood Prince",
    "Dobby is free. – Dobby, Harry Potter and the Chamber of Secrets",
    "You care so much you feel as though you will bleed to death with the pain of it. – Harry Potter, Harry Potter and the Order of the Phoenix",
    "The ones that love us never really leave us. – Sirius Black, Harry Potter and the Prisoner of Azkaban",
    "It is our choices that show what we truly are, far more than our abilities. – Albus Dumbledore, Harry Potter and the Chamber of Secrets",
    "Happiness can be found even in the darkest of times, if one only remembers to turn on the light. – Albus Dumbledore, Harry Potter and the Prisoner of Azkaban",
    "We’ve all got light and dark inside us. What matters is the part we choose to act on. That’s who we really are. – Sirius Black, Harry Potter and the Order of the Phoenix",
    "Do not pity the dead, Harry. Pity the living, and above all, those who live without love. – Albus Dumbledore, Harry Potter and the Deathly Hallows",
    "Working hard is important. But there’s something that matters even more, believing in yourself. – Harry Potter, Harry Potter and the Order of the Phoenix",
]

n = len(quotes)
if __name__=="__main__":
    for i in range(n):
        url = f"http://{HOSTNAME}/put"
        key_val = i % NO_OF_ENTRIES
        body = {
            "key": f"{KEY_FORMAT}_{key_val}",
            "value": quotes[i]
        }
        try:
            response = requests.post(url, json=body)
            if not (200 <= response.status_code <= 300):
                print("PUT K-V request failed: ", body["key"],"-", body["value"][0:7])
            print("KV ADDED")
        except Exception as e:
            print("Error", e)