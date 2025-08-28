
class Protocol:
    def __init__(self, socket):
        self._socket = socket

    def close_with(self, closure_to_close):
        """
        Double Dispatch
        Close this protocol's underlying socket using the given closer function

        :param closure_to_close: Function that accepts a socket and handles its closing logic
        """
        # Double Dispatch
        # Quiero mantener el socket privado para mantener la interaccion con el socket encapsulada en el protocolo
        # pero la responsabilidad del cierre y del registro de los logs en el servidor
        closure_to_close(self._socket)

    def wait_string(self):
        """
        Receive a string from the socket

        First, reads 2 bytes (big endian) that indicate the length, then reads the payload of that length
        Returns the decoded UTF-8 string
        """
        received_bytes = self.__recv_all(2)
        payload_size = (received_bytes[0] << 8) | received_bytes[1]
        payload = self.__recv_all(payload_size)
        return payload.decode("utf-8")

    def send_uint8(self, uint8):
        """
        Send a single unsigned byte (0-255) through the socket

        If it is not between 0 and 255 raise a ValueError exception
        """
        if not 0 <= uint8 <= 255:
            raise ValueError("Value must be an uint8 (0-255)")
        self.__send_all(bytes([uint8]))

    def __recv_all(self, n):
        """
        Read exactly n bytes from the socket

        Retries until the requested number of bytes is received
        Raises BrokenPipeError if the connection is closed unexpectedly
        """
        msg = b''
        while len(msg) < n:
            read = self._socket.recv(n - len(msg))
            if not read:
                raise BrokenPipeError
            msg += read
        return msg

    def __send_all(self, buf):
        """
        Send the entire contents of buf through the socket

        Retries until the entire buffer is sent
        Raises BrokenPipeError if the connection is closed unexpectedly
        """
        sz = 0
        while sz < len(buf):
            written = self._socket.send(buf[sz:])
            if not written:
                raise BrokenPipeError
            sz += written