import threading

from .utils import store_bets, load_bets, Bet

class ThreadSafeBetsStorage:
    """
    Thread-safe wrapper for store_bets and load_bets
    Reader Writer Lock, allows multiple readers concurrently
    This is not writer preference readers can continue entering while writers are waiting
    """
    def __init__(self):
        self._readers = 0
        self._writer = False
        self._lock = threading.Lock()
        self._readers_ok = threading.Condition(self._lock)
        self._writers_ok = threading.Condition(self._lock)

    def thread_safe_store_bets(self, bets: list[Bet]):
        """
        Persist the information of each bet in the STORAGE_FILEPATH file
        Thread-safe, does not handle poisoning
        """
        self.__acquire_write()
        try:
            store_bets(bets)
        finally:
            self.__release_write()

    def thread_safe_load_bets(self):
        """
        Loads the information all the bets in the STORAGE_FILEPATH file
        Thread-safe, does not handle poisoning
        """
        self.__acquire_read()
        try:
            yield from load_bets()
        finally:
            self.__release_read()

    def __acquire_read(self):
        """
        Acquire permission to read
        Waits if there is an active writer
        """
        with self._lock:
            while self._writer:
                self._readers_ok.wait()
            self._readers += 1

    def __release_read(self):
        """
        Release permission to read
        If was the last reader, notify one waiting writer
        """
        with self._lock:
            self._readers -= 1
            if self._readers == 0:
                self._writers_ok.notify()

    def __acquire_write(self):
        """
        Acquire permission to write
        Waits until there are no active readers or writers
        """
        with self._lock:
            while self._writer or self._readers > 0:
                self._writers_ok.wait()
            self._writer = True

    def __release_write(self):
        """
        Release permission to write
        Notifies all readers and writers
        """
        with self._lock:
            self._writer = False
            self._writers_ok.notify_all()
            self._readers_ok.notify_all()
