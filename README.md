## fbgdl
### facebook graph downloader

Downloads all the user entries for the Facebook graph. Put together in about
ten minutes.

it will create a sqlite3 database in ${CWD} with the table 'users'. the
fields for this are 

        id              64-bit unsigned integer
        name            string
        first           string
        last            string
        link            string
        username        string
        gender          string
        locale          string

if the application request limit is hit, the downloader will stall for one hour.
