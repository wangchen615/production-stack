import os
import logging

# Constants
GROUP = "production-stack.vllm.ai"
VERSION = "v1alpha1"
PLURAL = "loraadapters"
CRD_NAME = f"{PLURAL}.{GROUP}"

logger = logging.getLogger(__name__)

def load_config():
    """
    Load configuration from environment variables or defaults.
    
    Returns:
        dict: The configuration dictionary.
    """
    # Get the namespace to watch
    watch_namespace = os.environ.get("WATCH_NAMESPACE", "")
    
    # If watch_namespace contains commas, split it into a list of namespaces
    if "," in watch_namespace:
        watch_namespaces = [ns.strip() for ns in watch_namespace.split(",") if ns.strip()]
        
        if watch_namespaces:
            logger.info(f"Watching multiple namespaces: {watch_namespaces}")
            config = {
                "watch_namespaces": watch_namespaces
            }
        else:
            logger.info("No specific namespaces to watch, will watch all namespaces")
            config = {
                "watch_all_namespaces": True
            }
    elif watch_namespace:
        logger.info(f"Watching single namespace: {watch_namespace}")
        config = {
            "watch_namespace": watch_namespace
        }
    else:
        logger.info("No specific namespace to watch, will watch the current namespace")
        config = {
            "watch_current_namespace": True
        }
    
    # Get the resync period (in seconds)
    try:
        resync_period = int(os.environ.get("RESYNC_PERIOD", "300"))
    except ValueError:
        logger.warning("Invalid RESYNC_PERIOD, using default of 300 seconds")
        resync_period = 300
        
    config["resync_period"] = resync_period
    
    # Get the default adapter cache location
    config["default_cache_location"] = os.environ.get("DEFAULT_CACHE_LOCATION", "/tmp/adapters")
    
    # Get the label selector for pods with LoRA enabled
    config["lora_enabled_label"] = os.environ.get("LORA_ENABLED_LABEL", "lora-enabled=true")
    
    return config