import socket
import logging
import errno

from .application_protocol import ApplicationProtocol

class Server:
    @staticmethod
    def __try_close(a_socket, socket_name_to_log):
        """
        Attempt to gracefully close a given socket and log the result

        If the socket was already closed, for example by a signal, it will be silently ignored
        Any other errors will be logged
        """
        try:
            a_socket.close()
            logging.info(f'action: close_{socket_name_to_log} | result: success')
        except OSError as e:
            if e.errno == errno.EBADF:
                pass  # The socket was already closed by the graceful shutdown
            else:
                logging.error(f'action: close_{socket_name_to_log} | result: fail | error: {e}')

    def __init__(self, port, listen_backlog, clients_amount):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._clients_amount = int(clients_amount)
        self._client_count = 0
        self._client_protocol = None
        self._was_stopped = False
        self.winners_are_ready = False

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again

        If enough clients uploaded their bets, the winners are available
        """

        while not self._was_stopped:
            client_socket = self.__accept_new_connection()
            if client_socket:
                self._client_protocol = ApplicationProtocol(client_socket)
                self.__handle_client_connection()

            if self._client_count == self._clients_amount:
                logging.info('action: sorteo | result: success')
                self.winners_are_ready = True

    def graceful_shutdown(self, _signal_number, _current_stack_frame):
        """
        On a signal, gracefully shutdown the server

        Stops the main loop, closes the server socket and, if present, the active client socket
        """
        logging.info('action: graceful_shutdown | result: in_progress')

        self._was_stopped = True
        self.__try_close(self._server_socket, 'server_socket')
        if self._client_protocol:
            closure_to_close = lambda client_socket: self.__try_close(client_socket, 'client_socket')
            self._client_protocol.close_with(closure_to_close)

        logging.info('action: exit | result: success')

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket
        If operation completes without errors increment the client counter
        """
        try:
            self._client_protocol.wait_operation(self.winners_are_ready)
            self._client_count += 1
        except ValueError as e:
            if "A client tries to get the winners when are not ready" not in str(e):
                logging.error(f"action: error | result: fail | error: {e}")
        except Exception as e:
            logging.error(f"action: error | result: fail | error: {e}")
        finally:
            closure_to_close = lambda client_socket: self.__try_close(client_socket, 'client_socket')
            self._client_protocol.close_with(closure_to_close)

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
