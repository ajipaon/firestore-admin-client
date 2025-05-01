

# Example Data for Document Controller POST Request

To create a document using the document controller, you need to send a POST request to the following endpoint:

```
POST /api/collections/{collection}/documents
```

Where `{collection}` is the name of the Firestore collection where you want to create the document.

## Example Request

### URL
```
POST /api/collections/users/documents
```

### Headers
```
Content-Type: application/json
Authorization: Bearer your_auth_token_here
```

### Request Body
```json
{
  "name": "John Doe",
  "email": "john.doe@example.com",
  "age": 30,
  "address": {
    "street": "123 Main St",
    "city": "New York",
    "country": "USA",
    "postalCode": "10001"
  },
  "isActive": true,
  "tags": ["customer", "premium"]
}
```

## Notes

1. The system will automatically add the following fields to your document:
    - `createdAt`: Timestamp when the document was created
    - `createdBy`: User ID of the authenticated user who created the document

2. You can include any valid JSON data in your request body, including nested objects and arrays.

3. The response will include the ID of the newly created document:
   ```json
   {
     "success": true,
     "message": "Document created successfully",
     "data": {
       "id": "generated_document_id"
     }
   }
   ```

4. Authentication is required for this endpoint, so make sure to include a valid authentication token in your request.