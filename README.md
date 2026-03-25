This is Aggregate. A simple, CLI based RSS aggregator. It doesn't do anything fancy, it doesn't even parse HTML. All it does is retrieve RSS feeds, parses a given set of RSS feeds, and prints out the newest results

commands:
register [username] - create a new user
agg - starts the aggregation, pulling new results every minute
reset - clears all databases and information
users - lists the users, to include currently accessed user
addfeed [feed name] [feed url] - adds a feed to the feeds list, as well as subscribes active used to said feed
feeds - lists all feeds in the feeds database
follow [url] follow a feed that already exists in the feeds database
following - shows what feeds you are following
unfollow [url] - remove yourself as a follower of a feed
browse [number of feeds](optional, default 2) - retrieves the newest # of articles, based on the input given, default to 2 on no input, from all the feeds you are subscribed to
