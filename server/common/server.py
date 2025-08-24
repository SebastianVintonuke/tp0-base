import socket
import logging
import errno


class Server:
    @staticmethod
    def __try_close(a_socket, socket_name_to_log):
        try:
            a_socket.close()
            logging.info(f'action: close_{socket_name_to_log} | result: success')
        except OSError as e:
            if e.errno == errno.EBADF:
                pass  # The socket was already closed by the graceful shutdown
            else:
                logging.error(f'action: close_{socket_name_to_log} | result: fail | error: {e}')

    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._client_socket = None
        self._was_stopped = False

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        while not self._was_stopped:
            self._client_socket = self.__accept_new_connection()
            if self._client_socket:
                self.__handle_client_connection()

    def graceful_shutdown(self, _signal_number, _current_stack_frame):
        """
        On a signal, gracefully shutdown the server
        """
        logging.info('action: graceful_shutdown | result: in_progress')

        self._was_stopped = True
        self.__try_close(self._server_socket, 'server_socket')
        if self._client_socket:
            self.__try_close(self._client_socket, 'client_socket')

        logging.info('action: exit | result: success')

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = self._client_socket.recv(1024).rstrip().decode('utf-8')
            addr = self._client_socket.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')
            # TODO: Modify the send to avoid short-writes
            self._client_socket.send("{}\n".format(msg).encode('utf-8'))
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            self.__try_close(self._client_socket, 'client_socket')

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        try:
            c, addr = self._server_socket.accept()
            logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
            return c
        except OSError as e:
            if e.errno == errno.EBADF:
                return None  # The server socket was closed by the graceful shutdown
            else:
                logging.info(f'action: accept_connections | result: fail | error: {e}')
