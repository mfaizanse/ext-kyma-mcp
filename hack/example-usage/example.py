#!/usr/bin/env python3
"""
MCP Client to connect to a server via HTTP and list available tools.
"""
import asyncio
import httpx
import json
from fastmcp import Client
from fastmcp.client.transports import StreamableHttpTransport

def get_auth_object(cluster_json_file, authType):
    # authType: "token" or "cert"

    with open(cluster_json_file) as f:
      target_cluster = json.load(f)

    headers = {
      "x-target-k8s-certificate-authority-data": target_cluster["TARGET_CLUSTER_CLUSTER_CA_DATA"],
      "x-target-k8s-server": target_cluster["TARGET_CLUSTER_CLUSTER_URL"]
    }

    if authType == "token":
      headers["x-target-k8s-authorization"] = target_cluster["TARGET_CLUSTER_CLUSTER_AUTH_TOKEN"]
    elif authType == "cert":
      headers["x-target-k8s-client-certificate-data"] = target_cluster["TARGET_CLUSTER_CLUSTER_CLIENT_CERTIFICATE_DATA"]
      headers["x-target-k8s-client-key-data"] = target_cluster["TARGET_CLUSTER_CLUSTER_CLIENT_KEY_DATA"]
    return headers



async def connect_and_list_pods():   
    """Connect to the MCP server and list pods in a namespace.""" 
    server_url = "http://localhost:8085/mcp"
    cluster_json_file = "test-cluster.json"
    authType = "token"
    base64_enabled = False
    
    auth_object = get_auth_object(cluster_json_file, authType)
    
    custom_headers = {}
    if base64_enabled:
       custom_headers["Authorization"] = json.dumps(auth_object)
    else:
       custom_headers["Authorization"] = json.dumps(auth_object)

    try:
        # Define the tool parameters
        tool_name = "pods_list_in_namespace"
        tool_args = {
            "namespace": "kyma-system"
        }
        
        async with Client(
            transport=StreamableHttpTransport(
                server_url, 
                headers=custom_headers,
            ),
        ) as client:
            print(f"calling tool")
            # response = await client.call_tool(tool_name, tool_args, meta=custom_headers)
            response = await client.call_tool(tool_name, tool_args)
            print(response)
    except httpx.ConnectError:
        print(f"✗ Failed to connect to {server_url}")
    except Exception as e:
        print(f"✗ Error: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()


def main():
    """Main entry point."""
    asyncio.run(connect_and_list_pods())


if __name__ == "__main__":
    main()

