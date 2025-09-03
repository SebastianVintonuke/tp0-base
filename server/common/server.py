import socket
import logging
import errno
import threading

from .application_protocol import ApplicationProtocol
from .sync_utils import ThreadSafeBetsStorage

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
        self._was_stopped = False
        self._clients = []
        self.winners_are_ready = threading.Barrier(int(clients_amount))
        self.thread_safe_bets_storage = ThreadSafeBetsStorage()

    def run(self):
        """
        Multithreaded Server loop

        Server that accepts new connections and spawns a new thread
        for each client. Each client is handled independently.
        """

        while not self._was_stopped:
            self.__cleanup_client_threads()
            client_socket = self.__accept_new_connection()
            if client_socket:
                client_thread = threading.Thread(target=self.__handle_client_connection, args=(client_socket,))
                self._clients.append((client_thread, client_socket))
                client_thread.start()

    def graceful_shutdown(self, _signal_number, _current_stack_frame):
        """
        On a signal, gracefully shutdown the server

        Stops the main loop and closes the server socket finally close and join the clients
        """
        logging.info('action: graceful_shutdown | result: in_progress')

        self._was_stopped = True
        self.__try_close(self._server_socket, 'server_socket')

        for client_thread, client_socket in self._clients:
            self.__try_close(client_socket, 'client_socket')
            client_thread.join()

        logging.info('action: exit | result: success')

    def __handle_client_connection(self, client_socket):
        """
        Handle a client connection in a dedicated thread

        Read message, process it and close the socket
        """
        client_protocol = ApplicationProtocol(client_socket, self.thread_safe_bets_storage)

        try:
            client_protocol.wait_operation(self.winners_are_ready)

        except Exception as e:
            logging.error(f"action: error | result: fail | error: {e}")
        finally:
            closure_to_close = lambda sock: self.__try_close(sock, 'client_socket')
            client_protocol.close_with(closure_to_close)

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

    def __cleanup_client_threads(self):
        """
        Join the finished threads, if it finished, the socket was already closed
        To free resources we just need to join the threads and clean the references
        """
        alive_clients = []
        for thread, sock in self._clients:
            if thread.is_alive():
                alive_clients.append((thread, sock))
            else:
                thread.join()
        self._clients = alive_clients