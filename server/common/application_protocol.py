import logging

from .protocol import Protocol
from .utils import Bet, has_won

ERROR_CODE = 0x00
ACK_CODE = 0xFF

OPERATION_CODE_UPLOAD_BETS = 0x01
OPERATION_CODE_GET_WINNERS = 0x02

class ApplicationProtocol:
    def __init__(self, socket, bets_storage):
        self._protocol = Protocol(socket)
        self._bets_storage = bets_storage

    def close_with(self, closure_to_close):
        """
        Check Protocol.close_with method
        """
        self._protocol.close_with(closure_to_close)

    def wait_operation(self, winners_are_ready):
        """
        Waits for a client to send an operation code and processes it

        Depending on the received code, one of the following operations is done:
            - Upload bets (__operation_upload)
            - Get winners (__operation_get_winner)

        Args:
            winners_are_ready (Barrier): Indicates if the winners are ready

        Raises:
            ValueError: If a client tries to upload bets after winners are ready
                        or tries to get winners before they are ready
        """
        operation_code = self._protocol.wait_uint8()

        if operation_code == OPERATION_CODE_UPLOAD_BETS:
            self.__send_ack()
            self.__operation_upload()

        elif operation_code == OPERATION_CODE_GET_WINNERS:
            leader = winners_are_ready.wait()
            if leader == 0:
                logging.info('action: sorteo | result: success')

            self.__send_ack()
            self.__operation_get_winner()

        else:
            raise ValueError(f"Unknown operation code {operation_code}")

    def __operation_upload(self):
        """
        Receives all the bets from a client

        Receives batches of bets until a batch of size 0 is received
        Each batch is stored and acknowledged
        If an error occurs, try to send an error code (uint8 = 1)
        """
        try:
            batch_size = self.__wait_batch_size()
            while batch_size != 0:
                batch = self.__wait_batch(batch_size)
                logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(batch)}")
                self._bets_storage.thread_safe_store_bets(batch)
                self.__send_ack()
                batch_size = self.__wait_batch_size()
            self.__send_ack()

        except OSError as e:
            logging.error(f"action: apuesta_recibida | result: fail | error: {e}")
            self.__send_error()

    def __operation_get_winner(self):
        """
        Sends the winners to the client

        First get the agency ID by the client,
        then filters the winners from the agency
        Finally sends the winner's documents to the client
        """
        agency_id = int(self._protocol.wait_string())
        winners = []
        for bet in self._bets_storage.thread_safe_load_bets():
            if has_won(bet) and bet.agency == agency_id:
                winners.append(str(bet.document))
        self.__send_winners(winners)

    def __wait_batch(self, size):
        """
        Wait for a batch of bets from the client

        Reads `size` bets from the client connection and returns them as a list.
        """
        batch = []
        for i in range(0, size):
            bet = self.__wait_bet()
            batch.append(bet)
        return batch

    def __wait_batch_size(self):
        """
        Wait for the next batch size from the client

        Reads a single uint8 value that specifies the maximum number of bets in the next batch
        Returns 0 when the client signals EOF
        """
        return self._protocol.wait_uint8()

    def __wait_bet(self):
        """
        Wait for a bet from the client

        Receives and reconstructs a Bet object by reading its fields
        (first name, last name, document, birthdate, number) as strings
        """
        client_id = self._protocol.wait_string()
        first_name = self._protocol.wait_string()
        last_name = self._protocol.wait_string()
        document = self._protocol.wait_string()
        birthdate = self._protocol.wait_string()
        number = self._protocol.wait_string()
        return Bet(client_id, first_name, last_name, document, birthdate, number)

    def __send_winners(self, winners):
        """
        Sends the list of winners to the client

        First sends an uint8 with the number of winners,
        then sends each document as a string
        """
        self._protocol.send_uint8(len(winners))
        for i in range(0, len(winners)):
            self._protocol.send_string(winners[i])

    def __send_ack(self):
        """
        Send an acknowledgement (uint8 = 0) to the client

        Used to confirm:
            1. That a batch was received and stored successfully
            2. That the EOF was received
        """
        self._protocol.send_uint8(ACK_CODE)

    def __send_error(self):
        """
        Send an error code (uint8 = 1) to the client

        Used to notify the client that an error occurred in the protocol
        """
        self._protocol.send_uint8(ERROR_CODE)