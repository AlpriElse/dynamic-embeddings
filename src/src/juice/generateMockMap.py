f = open("mock_map_output", "a")

MB = 1 << 20
size = 5*MB

out = "1\n" * size

f.write(out)
f.close()

