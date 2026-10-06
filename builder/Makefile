

clean:
	CONTAINER=$$(sudo docker images --filter=reference="archbuild" -q); \
	sudo docker rmi $$CONTAINER

packages:
	./run-container.sh