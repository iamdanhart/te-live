# Adds 10 singers to the queue with varied names and song selections.
# Run the app without rate limiting: make run-noenforce

POST http://localhost:8080/signup
[FormParams]
name: Jamie
song: {"title":"Bohemian Rhapsody","artist":"Queen"}
song: {"title":"All Star","artist":"Smash Mouth"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Priya
song: {"title":"Anti-Hero","artist":"Taylor Swift"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Marcus
song: {"title":"Back In Black","artist":"AC/DC"}
song: {"title":"Basket Case","artist":"Green Day"}
song: {"title":"Barracuda","artist":"Heart"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Sofia
song: {"title":"Bad Romance","artist":"Lady Gaga"}
song: {"title":"Before He Cheats","artist":"Carrie Underwood"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Derek
song: {"title":"Billie Jean","artist":"Michael Jackson"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Nora
song: {"title":"9 to 5","artist":"Dolly Parton"}
song: {"title":"Brown Eyed Girl","artist":"Van Morrison"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Eli
song: {"title":"Born to Run","artist":"Bruce Springsteen"}
song: {"title":"Bulls On Parade","artist":"Rage Against The Machine"}
song: {"title":"Burning Down the House","artist":"Talking Heads"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Carmen
song: {"title":"Beat It","artist":"Michael Jackson"}
song: {"title":"Bust A Move","artist":"Young MC"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Theo
song: {"title":"Baby Got Back","artist":"Sir Mix-A-Lot"}
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Leila
song: {"title":"Ain't No Mountain High Enough","artist":"Marvin Gaye, Tammi Terrell"}
song: {"title":"Blister In The Sun","artist":"Violent Femmes"}
HTTP 200