#!/usr/bin/env python3

import kopf
import logging
from utils.logging import setup_logging
from utils.config import load_config
from controllers.adapter_controller import register_handlers

def main():
    """Main entry point for the LoRA adapter controller."""
    # Set up logging
    setup_logging()
    logger = logging.getLogger(__name__)
    
    # Load configuration
    config = load_config()
    
    # Log startup information
    logger.info("Starting LoRA Adapter controller")
    logger.info(f"Watching namespace(s): {config.get('watch_namespace', 'current namespace')}")
    
    # Register handlers and start the controller
    register_handlers()
    
    # Run the Kopf operator
    kopf.run()

if __name__ == "__main__":
    main()