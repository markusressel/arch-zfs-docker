

cleanup:
	CONTAINER=$$(sudo docker images --filter=reference="archbuild" -q); \
	sudo docker rmi $$CONTAINER

run:
	sudo docker buildx build --tag archbuild --build-arg VARIANT="" .
	sudo docker run -i -t --rm -v ~/tmp/zfs:/package archbuild