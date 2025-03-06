import logging
import os

def setup_logging():
    """Configure logging for the controller."""
    log_level = os.environ.get("LOG_LEVEL", "INFO").upper()
    
    # Map string log level to logging constants
    log_level_map = {
        "DEBUG": logging.DEBUG,
        "INFO": logging.INFO,
        "WARNING": logging.WARNING,
        "ERROR": logging.ERROR,
        "CRITICAL": logging.CRITICAL
    }
    
    # Set up basic logging
    logging.basicConfig(
        level=log_level_map.get(log_level, logging.INFO),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )